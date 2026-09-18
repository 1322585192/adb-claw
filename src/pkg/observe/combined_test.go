package observe

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/llm-net/adb-claw/pkg/adb"
)

type mockObserveCommander struct {
	png []byte
}

func (m *mockObserveCommander) Shell(args ...string) (*adb.Result, error) {
	return &adb.Result{}, nil
}

func (m *mockObserveCommander) ExecOut(args ...string) ([]byte, error) {
	if len(args) >= 1 && args[0] == "screencap" {
		return m.png, nil
	}
	return nil, fmt.Errorf("unexpected exec-out %v", args)
}

func (m *mockObserveCommander) RawCommand(args ...string) (*adb.Result, error) {
	return nil, fmt.Errorf("not implemented")
}

func TestObserveDefaultPathWritesFile(t *testing.T) {
	t.Setenv("TMPDIR", t.TempDir())
	cmd := &mockObserveCommander{png: solidPNG(80, 120)}
	result := Observe(cmd, ObserveOptions{Format: "jpeg"})
	if result.Screenshot == nil {
		t.Fatalf("screenshot missing: %v", result.Errors)
	}
	if result.Screenshot.Path == "" {
		t.Fatal("observe must write a file; console base64 is not allowed")
	}
	if _, err := os.Stat(result.Screenshot.Path); err != nil {
		t.Fatalf("expected screenshot file: %v", err)
	}
}

func TestObserveWritesJPEGFileNoUITree(t *testing.T) {
	t.Setenv("TMPDIR", t.TempDir())
	path := filepath.Join(t.TempDir(), "obs.jpg")
	cmd := &mockObserveCommander{png: solidPNG(1080, 2340)}

	result := Observe(cmd, ObserveOptions{
		Format:  "jpeg",
		Quality: 60,
		Path:    path,
	})
	if result.Screenshot == nil {
		t.Fatalf("screenshot missing: %v", result.Errors)
	}
	rawJSON, err := json.Marshal(result)
	if err != nil {
		t.Fatal(err)
	}
	s := string(rawJSON)
	if strings.Contains(s, "base64") {
		t.Errorf("JSON must never contain base64: %s", rawJSON)
	}
	if strings.Contains(s, `"ui"`) || strings.Contains(s, "elements") || strings.Contains(s, "state_id") {
		t.Errorf("observe must not return a UI tree: %s", rawJSON)
	}
	if result.Screenshot.DeviceWidth != 1080 {
		t.Errorf("device_width = %d, want 1080", result.Screenshot.DeviceWidth)
	}
}

func TestObserveScaledPreviewKeepsDeviceSize(t *testing.T) {
	t.Setenv("TMPDIR", t.TempDir())
	path := filepath.Join(t.TempDir(), "obs.jpg")
	cmd := &mockObserveCommander{png: solidPNG(1080, 2340)}

	result := Observe(cmd, ObserveOptions{
		MaxWidth: 540,
		Format:   "jpeg",
		Path:     path,
	})
	if result.Screenshot == nil {
		t.Fatalf("partial result: %+v errors=%v", result.Screenshot, result.Errors)
	}
	if result.Screenshot.Scale != 0.5 {
		t.Errorf("scale = %v, want 0.5", result.Screenshot.Scale)
	}
	if result.Screenshot.DeviceWidth != 1080 || result.Screenshot.DeviceHeight != 2340 {
		t.Errorf("device size changed: %dx%d", result.Screenshot.DeviceWidth, result.Screenshot.DeviceHeight)
	}
}
