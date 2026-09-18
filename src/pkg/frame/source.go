package frame

import (
	"bufio"
	"context"
	"crypto/md5"
	_ "embed"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/llm-net/adb-claw/pkg/adb"
)

//go:embed classes.dex
var dexData []byte

const deviceDEXPath = "/data/local/tmp/adbclaw-frame.dex"

// Options configure a persistent frame source.
type Options struct {
	Interval   time.Duration
	Width      int
	Quality    int
	LatestPath string
	ForceHigh  bool
	DisableDEX bool
	OnFrame    func(*Frame)
}

// Source streams the newest screen JPEG from a Frame DEX or screencap fallback.
type Source struct {
	cmd     *adb.Client
	opts    Options
	buf     *Buffer
	adapt   *Adaptive
	cancel  context.CancelFunc
	wg      sync.WaitGroup
	mu      sync.Mutex
	mode    string // dex | pull | stream
	stdin   io.WriteCloser
	proc    *exec.Cmd
	history []FrameMeta
}

// FrameMeta is lightweight history used to validate act(frame_seq).
type FrameMeta struct {
	Seq          uint32
	Hash         string
	CapturedAt   time.Time
	DeviceWidth  int
	DeviceHeight int
	Rotation     int
}

// Start launches a persistent frame source. DEX failure falls back to screencap.
func Start(parent context.Context, client *adb.Client, opts Options) (*Source, error) {
	if client == nil {
		return nil, fmt.Errorf("frame source requires an adb client")
	}
	if opts.Interval <= 0 {
		opts.Interval = time.Duration(defaultIntervalMs) * time.Millisecond
	}
	if opts.LatestPath == "" {
		opts.LatestPath = DefaultLatestPath()
	}
	if opts.ForceHigh && opts.Width == 0 {
		opts.Width = WidthHigh
		opts.Quality = QualityHigh
	}
	ctx, cancel := context.WithCancel(parent)
	s := &Source{
		cmd:    client,
		opts:   opts,
		buf:    NewBuffer(),
		adapt:  NewAdaptive(opts.Width, opts.Quality),
		cancel: cancel,
	}

	if !opts.DisableDEX {
		if err := EnsureDEX(client); err == nil {
			if err := s.startDEX(ctx); err == nil {
				s.mode = "dex"
				return s, nil
			}
		}
	}

	s.mode = "pull"
	s.wg.Add(1)
	go s.pullLoop(ctx)
	return s, nil
}

// Mode is dex, pull, or stream.
func (s *Source) Mode() string { return s.mode }

// Resolution is the current encode width target.
func (s *Source) Resolution() int { return s.adapt.Width() }

// Buffer exposes the capacity-1 latest-frame buffer.
func (s *Source) Buffer() *Buffer { return s.buf }

// Latest returns the newest frame.
func (s *Source) Latest() *Frame { return s.buf.Latest() }

// WaitAfter waits for a visual change after seq/hash.
func (s *Source) WaitAfter(seq uint32, hash string, timeout time.Duration) *Frame {
	if timeout <= 0 {
		timeout = 2 * time.Second
	}
	return s.buf.WaitAfter(seq, hash, timeout)
}

// Lookup returns metadata for a previously seen frame_seq.
func (s *Source) Lookup(seq uint32) *FrameMeta {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := len(s.history) - 1; i >= 0; i-- {
		if s.history[i].Seq == seq {
			meta := s.history[i]
			return &meta
		}
	}
	return nil
}

// Close stops the device process and reader.
func (s *Source) Close() error {
	s.cancel()
	s.mu.Lock()
	if s.stdin != nil {
		_ = s.stdin.Close()
		s.stdin = nil
	}
	if s.proc != nil && s.proc.Process != nil {
		_ = s.proc.Process.Kill()
	}
	s.mu.Unlock()
	s.wg.Wait()
	return nil
}

func (s *Source) startDEX(ctx context.Context) error {
	args := s.cmd.BaseArgs()
	args = append(args, "exec-out",
		"CLASSPATH="+deviceDEXPath,
		"app_process", "/", "ADBClawBridge",
		"--interval", fmt.Sprintf("%d", s.opts.Interval.Milliseconds()),
		"--width", fmt.Sprintf("%d", s.adapt.Width()),
		"--quality", fmt.Sprintf("%d", s.adapt.Quality()),
	)
	cmd := exec.CommandContext(ctx, s.cmd.ADBPath, args...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return err
	}
	if err := cmd.Start(); err != nil {
		return err
	}

	ready := make(chan error, 1)
	go func() {
		sc := bufio.NewScanner(stderr)
		for sc.Scan() {
			line := sc.Text()
			if strings.Contains(line, "ready") {
				ready <- nil
				return
			}
			low := strings.ToLower(line)
			if strings.Contains(low, "classnotfound") || strings.Contains(low, "connect failed") {
				ready <- fmt.Errorf("%s", line)
				return
			}
		}
		ready <- fmt.Errorf("frame dex exited before ready")
	}()

	select {
	case err := <-ready:
		if err != nil {
			_ = cmd.Process.Kill()
			return err
		}
	case <-time.After(1500 * time.Millisecond):
		_ = cmd.Process.Kill()
		return fmt.Errorf("frame dex start timeout")
	}

	s.mu.Lock()
	s.proc = cmd
	s.stdin = stdin
	s.mu.Unlock()

	s.wg.Add(1)
	go s.readLoop(ctx, stdout)
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		_ = cmd.Wait()
	}()
	return nil
}

func (s *Source) readLoop(ctx context.Context, r io.Reader) {
	defer s.wg.Done()
	for {
		if ctx.Err() != nil {
			return
		}
		f, err := ReadFrame(r)
		if err != nil {
			if ctx.Err() != nil || err == io.EOF {
				return
			}
			return
		}
		f.Source = "dex"
		s.accept(f)
	}
}

func (s *Source) pullLoop(ctx context.Context) {
	defer s.wg.Done()
	ticker := time.NewTicker(s.opts.Interval)
	defer ticker.Stop()
	var seq uint32
	capture := func() {
		seq++
		f, err := captureFallback(s.cmd, seq, s.adapt.Width(), s.adapt.Quality())
		if err != nil {
			return
		}
		s.accept(f)
	}
	capture()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			capture()
		}
	}
}

func (s *Source) accept(f *Frame) {
	if f == nil {
		return
	}
	s.buf.Put(f)
	s.record(f)
	if s.opts.LatestPath != "" && !f.Duplicate {
		_ = WriteAtomic(s.opts.LatestPath, f.JPEG)
	}
	if s.opts.OnFrame != nil {
		s.opts.OnFrame(f)
	}
	if s.adapt.Observe(f.Age().Milliseconds(), f.TransferMs) {
		s.applyResolution()
	}
}

func (s *Source) record(f *Frame) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.history = append(s.history, FrameMeta{
		Seq:          f.Seq,
		Hash:         f.Hash,
		CapturedAt:   f.CapturedAt(),
		DeviceWidth:  int(f.DeviceWidth),
		DeviceHeight: int(f.DeviceHeight),
		Rotation:     int(f.Rotation),
	})
	if len(s.history) > 32 {
		s.history = s.history[len(s.history)-32:]
	}
}

func (s *Source) applyResolution() {
	s.mu.Lock()
	stdin := s.stdin
	s.mu.Unlock()
	if stdin == nil {
		return
	}
	_, _ = fmt.Fprintf(stdin, "RES %d %d\n", s.adapt.Width(), s.adapt.Quality())
}

// EnsureDEX pushes the embedded Frame DEX when the on-device copy differs.
func EnsureDEX(client *adb.Client) error {
	if len(dexData) == 0 {
		return fmt.Errorf("embedded frame DEX is empty")
	}
	localMD5 := fmt.Sprintf("%x", md5.Sum(dexData))
	result, err := client.Shell("md5sum", deviceDEXPath)
	if err == nil && result.ExitCode == 0 {
		parts := strings.Fields(strings.TrimSpace(result.Stdout))
		if len(parts) > 0 && parts[0] == localMD5 {
			return nil
		}
	}
	tmpFile := filepath.Join(os.TempDir(), "adbclaw-frame.dex")
	if err := os.WriteFile(tmpFile, dexData, 0644); err != nil {
		return fmt.Errorf("write temp DEX: %w", err)
	}
	defer os.Remove(tmpFile)
	result, err = client.RawCommand("push", tmpFile, deviceDEXPath)
	if err != nil {
		return fmt.Errorf("push frame DEX: %w", err)
	}
	if result.ExitCode != 0 {
		msg := strings.TrimSpace(result.Stderr)
		if msg == "" {
			msg = fmt.Sprintf("adb push exit %d", result.ExitCode)
		}
		return fmt.Errorf("push frame DEX: %s", msg)
	}
	return nil
}

// EmbeddedDEX reports whether a DEX blob is compiled in (used by doctor/tests).
func EmbeddedDEX() []byte { return dexData }
