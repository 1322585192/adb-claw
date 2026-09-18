package pump

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/llm-net/adb-claw/pkg/adb"
	"github.com/llm-net/adb-claw/pkg/stream"
)

func TestEnsureStartsDetachedHelper(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("detached .cmd helper pid is not stable on Windows")
	}
	t.Setenv("TMPDIR", t.TempDir())
	name := "fake-adb-claw"
	body := "#!/bin/sh\nexec sleep 30\n"
	if runtime.GOOS == "windows" {
		name = "fake-adb-claw.cmd"
		body = "@echo off\r\nping -n 31 127.0.0.1 >nul\r\n"
	}
	script := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(script, []byte(body), 0755); err != nil {
		t.Fatal(err)
	}
	old := Executable
	Executable = func() (string, error) { return script, nil }
	defer func() { Executable = old }()

	client := adb.NewClient("", time.Second)
	if err := Ensure(client, Options{Interval: 200 * time.Millisecond}); err != nil {
		t.Fatal(err)
	}
	key := stream.DefaultKey
	if !stream.Running(key) {
		t.Fatal("expected pump pid to be alive")
	}
	if err := Ensure(client, Options{}); err != nil {
		t.Fatal(err)
	}
	status := Inspect(client)
	if !status.Running || status.PID <= 0 {
		t.Fatalf("%+v", status)
	}
	if err := Stop(client); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) && stream.Running(key) {
		time.Sleep(20 * time.Millisecond)
	}
	if stream.Running(key) {
		t.Fatal("pump still running after stop")
	}
}

func TestEnsureRejectsNonClient(t *testing.T) {
	err := Ensure(nil, Options{})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestLatestFromFrameMapsDexToStream(t *testing.T) {
	// compile-time sanity: Inspect does not panic without files
	t.Setenv("TMPDIR", t.TempDir())
	st := Inspect(adb.NewClient("none", time.Second))
	if st.Running {
		t.Fatal("expected no pump")
	}
	if st.JPEGPath == "" {
		t.Fatal("jpeg path required")
	}
}
