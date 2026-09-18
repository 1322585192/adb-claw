package observe

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/llm-net/adb-claw/pkg/adb"
)

type recordingCommander struct {
	calls [][]string
	xml   string
}

func (r *recordingCommander) Shell(args ...string) (*adb.Result, error) {
	r.calls = append(r.calls, append([]string{"shell"}, args...))
	if len(args) == 1 && strings.Contains(args[0], "uiautomator dump") {
		if !strings.Contains(args[0], "cat ") || !strings.Contains(args[0], "rm -f") {
			return &adb.Result{ExitCode: 1, Stderr: "unexpected script: " + args[0]}, nil
		}
		return &adb.Result{Stdout: r.xml}, nil
	}
	if len(args) >= 2 && args[0] == "sh" && args[1] == "-c" {
		if !strings.Contains(args[2], "uiautomator dump") || !strings.Contains(args[2], "cat ") || !strings.Contains(args[2], "rm -f") {
			return &adb.Result{ExitCode: 1, Stderr: "unexpected script: " + args[2]}, nil
		}
		return &adb.Result{Stdout: r.xml}, nil
	}
	return &adb.Result{}, nil
}

func (r *recordingCommander) ExecOut(args ...string) ([]byte, error) {
	r.calls = append(r.calls, append([]string{"exec-out"}, args...))
	return nil, fmt.Errorf("not used")
}

func (r *recordingCommander) RawCommand(args ...string) (*adb.Result, error) {
	r.calls = append(r.calls, append([]string{"raw"}, args...))
	return nil, fmt.Errorf("not used")
}

func TestDumpUITreeSingleShell(t *testing.T) {
	t.Setenv("TMPDIR", t.TempDir())
	cmd := &recordingCommander{xml: sampleXML}
	tree, err := DumpUITreeOpts(cmd, DumpOptions{Profile: true, Save: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(tree.Elements) != 4 {
		t.Fatalf("elements = %d, want 4", len(tree.Elements))
	}
	if tree.StateID == "" {
		t.Fatal("expected state_id")
	}
	if tree.Profile == nil || tree.Profile.ADBCalls != 1 {
		t.Fatalf("profile adb_calls = %+v, want 1", tree.Profile)
	}
	if len(cmd.calls) != 1 {
		t.Fatalf("ADB calls = %d, want 1: %v", len(cmd.calls), cmd.calls)
	}
	if !strings.Contains(strings.Join(cmd.calls[0], " "), "uiautomator dump") {
		t.Errorf("expected combined dump script, got %v", cmd.calls[0])
	}

	snap, err := LoadSnapshot("", time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if snap.StateID != tree.StateID {
		t.Errorf("saved state_id = %s, want %s", snap.StateID, tree.StateID)
	}
}

func TestDumpUITreeCompressedScript(t *testing.T) {
	t.Setenv("TMPDIR", t.TempDir())
	cmd := &recordingCommander{xml: sampleXML}
	if _, err := DumpUITreeOpts(cmd, DumpOptions{Compressed: true}); err != nil {
		t.Fatal(err)
	}
	script := strings.Join(cmd.calls[0], " ")
	if !strings.Contains(script, "dump --compressed") {
		t.Errorf("script = %q, want --compressed", script)
	}
}

func TestDumpUITreeRealtimeMode(t *testing.T) {
	t.Setenv("TMPDIR", t.TempDir())
	cmd := &recordingCommander{xml: sampleXML}
	tree, err := DumpUITreeOpts(cmd, DumpOptions{Mode: UIModeRealtime})
	if err != nil {
		t.Fatal(err)
	}
	for _, el := range tree.Elements {
		if el.Class != "" {
			t.Errorf("realtime profile should omit class, got %q", el.Class)
		}
		if el.Handle == "" {
			t.Error("expected handle")
		}
	}
}
