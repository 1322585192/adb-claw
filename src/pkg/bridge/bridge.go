package bridge

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/llm-net/adb-claw/pkg/adb"
	"github.com/llm-net/adb-claw/pkg/monitor"
	"github.com/llm-net/adb-claw/pkg/observe"
)

// Event is a UI or lifecycle notification from the device bridge.
type Event struct {
	Type   string          `json:"type"`
	Source string          `json:"source"`
	Tree   *observe.UITree `json:"tree,omitempty"`
	Raw    string          `json:"raw,omitempty"`
	Err    error           `json:"-"`
}

// Process is a long-lived UI watcher.
type Process struct {
	cancel context.CancelFunc
	events chan Event
	wg     sync.WaitGroup
	mode   string
}

// Options control how the bridge is started.
type Options struct {
	Debounce time.Duration
	Poll     time.Duration
}

// Start launches the best available UI watcher: DEX bridge, uiautomator events, or dump poll.
func Start(parent context.Context, client *adb.Client, opts Options) (*Process, error) {
	if opts.Debounce <= 0 {
		opts.Debounce = 80 * time.Millisecond
	}
	if opts.Poll <= 0 {
		opts.Poll = 750 * time.Millisecond
	}
	ctx, cancel := context.WithCancel(parent)
	p := &Process{
		cancel: cancel,
		events: make(chan Event, 16),
	}

	if err := monitor.EnsureDEX(client); err == nil {
		if src, err := startDEX(ctx, client, opts.Debounce); err == nil {
			p.mode = "dex"
			p.wg.Add(1)
			go p.forward(src)
			return p, nil
		}
	}

	if src, err := startEvents(ctx, client, opts.Debounce); err == nil {
		p.mode = "events"
		p.wg.Add(1)
		go p.consumeWakes(ctx, src, opts.Debounce, client)
		return p, nil
	}

	p.mode = "poll"
	p.wg.Add(1)
	go p.poll(ctx, client, opts.Poll)
	return p, nil
}

func (p *Process) Events() <-chan Event { return p.events }
func (p *Process) Mode() string         { return p.mode }

func (p *Process) Stop() {
	p.cancel()
	p.wg.Wait()
}

func (p *Process) forward(src <-chan Event) {
	defer p.wg.Done()
	defer close(p.events)
	for ev := range src {
		p.events <- ev
	}
}

func (p *Process) consumeWakes(ctx context.Context, wakes <-chan struct{}, debounce time.Duration, client *adb.Client) {
	defer p.wg.Done()
	defer close(p.events)
	var last string
	dump := func() {
		tree, err := observe.DumpUITreeOpts(client, observe.DumpOptions{Mode: observe.UIModeRealtime, Save: true})
		if err != nil {
			p.events <- Event{Type: "error", Source: "events", Err: err, Raw: err.Error()}
			return
		}
		h := tree.Hash()
		if h == last {
			return
		}
		last = h
		p.events <- Event{Type: "ui", Source: "events", Tree: tree}
	}

	timer := time.NewTimer(debounce)
	if !timer.Stop() {
		<-timer.C
	}
	armed := false
	for {
		select {
		case <-ctx.Done():
			return
		case _, ok := <-wakes:
			if !ok {
				return
			}
			if armed && !timer.Stop() {
				select {
				case <-timer.C:
				default:
				}
			}
			timer.Reset(debounce)
			armed = true
		case <-timer.C:
			armed = false
			dump()
		}
	}
}

func (p *Process) poll(ctx context.Context, client *adb.Client, interval time.Duration) {
	defer p.wg.Done()
	defer close(p.events)
	var last string
	tick := time.NewTicker(interval)
	defer tick.Stop()
	dump := func() {
		tree, err := observe.DumpUITreeOpts(client, observe.DumpOptions{Mode: observe.UIModeRealtime, Save: true})
		if err != nil {
			p.events <- Event{Type: "error", Source: "poll", Err: err, Raw: err.Error()}
			return
		}
		h := tree.Hash()
		if h == last {
			return
		}
		last = h
		p.events <- Event{Type: "ui", Source: "poll", Tree: tree}
	}
	dump()
	for {
		select {
		case <-ctx.Done():
			return
		case <-tick.C:
			dump()
		}
	}
}

func startDEX(ctx context.Context, client *adb.Client, debounce time.Duration) (<-chan Event, error) {
	args := client.BaseArgs()
	args = append(args, "shell",
		"CLASSPATH=/data/local/tmp/adbclaw-monitor.dex",
		"app_process", "/", "ADBClawBridge",
		"--debounce", fmt.Sprintf("%d", debounce.Milliseconds()),
	)
	cmd := exec.CommandContext(ctx, client.ADBPath, args...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, err
	}
	if err := cmd.Start(); err != nil {
		return nil, err
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
		ready <- fmt.Errorf("bridge dex exited before ready")
	}()

	select {
	case err := <-ready:
		if err != nil {
			_ = cmd.Process.Kill()
			return nil, err
		}
	case <-time.After(3 * time.Second):
		_ = cmd.Process.Kill()
		return nil, fmt.Errorf("bridge dex start timeout")
	}

	out := make(chan Event, 8)
	go func() {
		defer close(out)
		defer cmd.Wait()
		sc := bufio.NewScanner(stdout)
		sc.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
		for sc.Scan() {
			line := sc.Text()
			var raw map[string]json.RawMessage
			if json.Unmarshal([]byte(line), &raw) != nil {
				continue
			}
			out <- Event{Type: "ui", Source: "dex", Raw: line}
		}
	}()
	return out, nil
}

func startEvents(ctx context.Context, client *adb.Client, _ time.Duration) (<-chan struct{}, error) {
	args := client.BaseArgs()
	args = append(args, "shell", "uiautomator", "events")
	cmd := exec.CommandContext(ctx, client.ADBPath, args...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	if err := cmd.Start(); err != nil {
		return nil, err
	}

	wakes := make(chan struct{}, 1)
	go func() {
		defer close(wakes)
		defer cmd.Wait()
		sc := bufio.NewScanner(stdout)
		for sc.Scan() {
			select {
			case wakes <- struct{}{}:
			default:
			}
		}
	}()
	return wakes, nil
}
