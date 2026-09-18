package pump

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"syscall"
	"time"

	"github.com/llm-net/adb-claw/pkg/adb"
	"github.com/llm-net/adb-claw/pkg/observe"
	"github.com/llm-net/adb-claw/pkg/stream"
)

// Executable is os.Executable, overridable in tests.
var Executable = os.Executable

func init() {
	observe.EnsureStream = func(cmd adb.Commander, opts observe.ObserveOptions) error {
		return Ensure(cmd, Options{
			Width:   opts.MaxWidth,
			Quality: opts.Quality,
		})
	}
}

// Ensure starts a detached pump for the device when one is not already alive.
func Ensure(cmd adb.Commander, opts Options) error {
	client, ok := cmd.(*adb.Client)
	if !ok || client == nil {
		return fmt.Errorf("live stream requires an adb client")
	}
	if opts.Serial == "" {
		opts.Serial = client.Serial
	}
	key := stream.ResolveKey(client)
	lock, err := lockFile(stream.LockPath(key))
	if err != nil {
		return err
	}
	defer unlockFile(lock)

	if stream.Running(key) {
		if stream.Fresh(key, 3*time.Second) {
			return nil
		}
		// Process is up but has not published a frame yet; let the caller wait.
		return nil
	}
	return startDaemon(client, key, opts)
}

func startDaemon(client *adb.Client, key string, opts Options) error {
	exe, err := Executable()
	if err != nil {
		return fmt.Errorf("locate adb-claw: %w", err)
	}
	if opts.Interval <= 0 {
		opts.Interval = 250 * time.Millisecond
	}
	if opts.Quality <= 0 {
		opts.Quality = 60
	}
	args := []string{"pump", "--interval", strconv.FormatInt(opts.Interval.Milliseconds(), 10),
		"--quality", strconv.Itoa(opts.Quality),
		"--width", strconv.Itoa(opts.Width)}
	if client.Serial != "" {
		args = append([]string{"-s", client.Serial}, args...)
	}
	if err := os.MkdirAll(stream.Dir(key), 0755); err != nil {
		return err
	}
	logFile, err := os.OpenFile(stream.LogPath(key), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return err
	}
	defer logFile.Close()

	cmd := exec.Command(exe, args...)
	cmd.Stdout = logFile
	cmd.Stderr = logFile
	cmd.Stdin = nil
	cmd.Env = os.Environ()
	detach(cmd)
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start pump: %w", err)
	}
	pid := cmd.Process.Pid
	_ = stream.WritePID(key, pid)
	_ = cmd.Process.Release()
	return nil
}

// Stop terminates the detached pump for the device.
func Stop(cmd adb.Commander) error {
	key := stream.ResolveKey(cmd)
	lock, err := lockFile(stream.LockPath(key))
	if err != nil {
		return err
	}
	defer unlockFile(lock)
	return stopLocked(key)
}

func stopLocked(key string) error {
	pid, err := stream.ReadPID(key)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	process, err := os.FindProcess(pid)
	if err == nil && stream.PIDAlive(pid) {
		_ = process.Signal(syscall.SIGTERM)
		deadline := time.Now().Add(1500 * time.Millisecond)
		for time.Now().Before(deadline) && stream.PIDAlive(pid) {
			time.Sleep(30 * time.Millisecond)
		}
		if stream.PIDAlive(pid) {
			_ = process.Signal(syscall.SIGKILL)
		}
	}
	stream.RemovePID(key)
	return nil
}

// Inspect returns whether the pump is alive and the latest sidecar.
func Inspect(cmd adb.Commander) Status {
	key := stream.ResolveKey(cmd)
	status := Status{
		Key:      key,
		Serial:   stream.ClientSerial(cmd),
		JPEGPath: stream.JPEGPath(key),
		LogPath:  stream.LogPath(key),
	}
	if pid, err := stream.ReadPID(key); err == nil {
		status.PID = pid
		status.Running = stream.PIDAlive(pid)
		if !status.Running {
			stream.RemovePID(key)
			status.PID = 0
		}
	}
	if latest, err := stream.Read(key); err == nil {
		status.Latest = latest
		status.AgeMs = stream.Age(latest).Milliseconds()
	}
	return status
}
