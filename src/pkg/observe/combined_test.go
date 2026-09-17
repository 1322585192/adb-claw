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
	if len(args) >= 2 && args[0] == "uiautomator" && args[1] == "dump" {
		return &adb.Result{Stdout: "UI hierchary dumped to: " + args[2] + "\n"}, nil
	}
	if len(args) >= 1 && args[0] == "cat" {
		return &adb.Result{Stdout: sampleXML}, nil
	}
	if len(args) >= 1 && args[0] == "rm" {
		return &adb.Result{}, nil
	}
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
	// Isolate DefaultObservePath by using a unique format path via -- we pass Path.
	// Default path writes are covered by Observe() when Path is empty; skip if we cannot
	// control TMPDIR safely alongside parallel tests.
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

func TestObserveWritesJPEGFileKeepsDeviceCoords(t *testing.T) {
	path := filepath.Join(t.TempDir(), "obs.jpg")
	cmd := &mockObserveCommander{png: solidPNG(1080, 2340)}

	result := Observe(cmd, ObserveOptions{
		Format:  "jpeg",
		Quality: 70,
		Path:    path,
	})
	if result.Screenshot == nil {
		t.Fatalf("screenshot missing: %v", result.Errors)
	}
	if result.UI == nil {
		t.Fatalf("ui tree missing: %v", result.Errors)
	}
	rawJSON, err := json.Marshal(result.Screenshot)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(rawJSON), "base64") {
		t.Errorf("JSON must never contain base64: %s", rawJSON)
	}
	if result.Screenshot.Scale != 1 {
		t.Errorf("scale = %v, want 1", result.Screenshot.Scale)
	}
	if result.Screenshot.DeviceWidth != 1080 {
		t.Errorf("device_width = %d, want 1080", result.Screenshot.DeviceWidth)
	}

	el := result.UI.Elements[0]
	if el.Center.X != 540 || el.Center.Y != 150 {
		t.Errorf("UI center = (%d,%d), want device pixels (540,150)", el.Center.X, el.Center.Y)
	}
	if result.UI.Package != "com.example" {
		t.Errorf("package = %q", result.UI.Package)
	}
}

func TestObserveScaledPreviewDoesNotChangeUICoords(t *testing.T) {
	path := filepath.Join(t.TempDir(), "obs.jpg")
	cmd := &mockObserveCommander{png: solidPNG(1080, 2340)}

	result := Observe(cmd, ObserveOptions{
		MaxWidth: 540,
		Format:   "jpeg",
		Path:     path,
	})
	if result.Screenshot == nil || result.UI == nil {
		t.Fatalf("partial result: %+v errors=%v", result.Screenshot, result.Errors)
	}
	if result.Screenshot.Scale != 0.5 {
		t.Errorf("scale = %v, want 0.5", result.Screenshot.Scale)
	}
	found := result.UI.FindByText("Login")
	if len(found) != 1 {
		t.Fatalf("expected 1 Login element, got %d", len(found))
	}
	el := found[0]
	if el.Center.X != 540 || el.Center.Y != 550 {
		t.Errorf("Login center = (%d,%d), want device pixels (540,550) not preview pixels", el.Center.X, el.Center.Y)
	}
	if len(result.Errors) > 0 {
		t.Errorf("unexpected errors: %v", result.Errors)
	}
}
