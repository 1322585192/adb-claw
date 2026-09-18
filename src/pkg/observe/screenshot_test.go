package observe

import (
	"bytes"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/llm-net/adb-claw/pkg/adb"
)

type mockCaptureCommander struct {
	png []byte
}

func (m *mockCaptureCommander) Shell(args ...string) (*adb.Result, error) {
	return &adb.Result{}, nil
}

func (m *mockCaptureCommander) ExecOut(args ...string) ([]byte, error) {
	if len(args) >= 1 && args[0] == "screencap" {
		return m.png, nil
	}
	return nil, fmt.Errorf("unexpected exec-out %v", args)
}

func (m *mockCaptureCommander) RawCommand(args ...string) (*adb.Result, error) {
	return nil, fmt.Errorf("not implemented")
}

func solidPNG(width, height int) []byte {
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	// Large color blocks, similar to a real UI screenshot (JPEG compresses these well).
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.Set(x, y, color.RGBA{
				R: uint8((x / 80) * 40),
				G: uint8((y / 120) * 30),
				B: 160,
				A: 255,
			})
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		panic(err)
	}
	return buf.Bytes()
}

func TestCaptureScreenshotJPEGWritesFileNoBase64(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "observe.jpg")
	cmd := &mockCaptureCommander{png: solidPNG(1080, 2340)}

	result, err := CaptureScreenshot(cmd, CaptureOptions{
		Format: "jpeg",
		Path:   path,
	})
	if err != nil {
		t.Fatalf("CaptureScreenshot: %v", err)
	}
	if result.Format != "jpeg" {
		t.Errorf("format = %q, want jpeg", result.Format)
	}
	rawJSON, err := json.Marshal(result)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(rawJSON), "base64") {
		t.Errorf("JSON must never contain base64: %s", rawJSON)
	}
	if result.DeviceWidth != 1080 || result.DeviceHeight != 2340 {
		t.Errorf("device size = %dx%d, want 1080x2340", result.DeviceWidth, result.DeviceHeight)
	}
	if result.ImageWidth != 1080 || result.ImageHeight != 2340 {
		t.Errorf("image size = %dx%d, want 1080x2340", result.ImageWidth, result.ImageHeight)
	}
	if result.Scale != 1 {
		t.Errorf("scale = %v, want 1", result.Scale)
	}
	if result.Path != path {
		t.Errorf("path = %q, want %q", result.Path, path)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read output: %v", err)
	}
	if len(data) < 2 || data[0] != 0xFF || data[1] != 0xD8 {
		t.Fatal("output is not a JPEG")
	}
	rawPixels := 1080 * 2340 * 4
	if len(data) >= rawPixels {
		t.Errorf("jpeg (%d bytes) should be far smaller than raw RGBA (%d bytes)", len(data), rawPixels)
	}
	cfg, err := jpeg.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("jpeg decode: %v", err)
	}
	if cfg.Width != 1080 || cfg.Height != 2340 {
		t.Errorf("jpeg config = %dx%d, want 1080x2340", cfg.Width, cfg.Height)
	}
}

func TestCaptureScreenshotEmptyPathWritesDefaultFile(t *testing.T) {
	t.Setenv("TMPDIR", t.TempDir())
	cmd := &mockCaptureCommander{png: solidPNG(100, 200)}
	result, err := CaptureScreenshot(cmd, CaptureOptions{Format: "jpeg"})
	if err != nil {
		t.Fatalf("CaptureScreenshot: %v", err)
	}
	if result.Path == "" {
		t.Fatal("expected a file path; console base64 is not allowed")
	}
	if _, err := os.Stat(result.Path); err != nil {
		t.Fatalf("expected screenshot file: %v", err)
	}
	rawJSON, err := json.Marshal(result)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(rawJSON), "base64") {
		t.Errorf("JSON must never contain base64: %s", rawJSON)
	}
}

func TestCaptureScreenshotWidthScaleMetadata(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "preview.jpg")
	cmd := &mockCaptureCommander{png: solidPNG(1080, 2340)}

	result, err := CaptureScreenshot(cmd, CaptureOptions{
		MaxWidth: 540,
		Format:   "jpeg",
		Path:     path,
	})
	if err != nil {
		t.Fatalf("CaptureScreenshot: %v", err)
	}
	if result.DeviceWidth != 1080 || result.DeviceHeight != 2340 {
		t.Errorf("device size changed to %dx%d; must stay 1080x2340", result.DeviceWidth, result.DeviceHeight)
	}
	if result.ImageWidth != 540 || result.ImageHeight != 1170 {
		t.Errorf("image size = %dx%d, want 540x1170", result.ImageWidth, result.ImageHeight)
	}
	if result.Scale != 0.5 {
		t.Errorf("scale = %v, want 0.5", result.Scale)
	}
}

func TestCaptureScreenshotInvalidFormat(t *testing.T) {
	cmd := &mockCaptureCommander{png: solidPNG(10, 10)}
	_, err := CaptureScreenshot(cmd, CaptureOptions{Format: "gif"})
	if err == nil || !strings.Contains(err.Error(), "unsupported screenshot format") {
		t.Fatalf("expected format error, got %v", err)
	}
}

func TestNormalizeDefaults(t *testing.T) {
	opts, err := normalizeCaptureOptions(CaptureOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if opts.Format != "jpeg" {
		t.Errorf("format = %q, want jpeg", opts.Format)
	}
	if opts.Quality != DefaultJPEGQuality {
		t.Errorf("quality = %d, want %d", opts.Quality, DefaultJPEGQuality)
	}
	if !strings.HasSuffix(opts.Path, "adb-claw-screenshot.jpg") {
		t.Errorf("empty path should default to screenshot file, got %q", opts.Path)
	}
}

func TestDefaultObservePath(t *testing.T) {
	p := DefaultObservePath("jpeg")
	if !strings.HasSuffix(p, "adb-claw-observe.jpg") {
		t.Errorf("default jpeg path = %q", p)
	}
	p = DefaultObservePath("png")
	if !strings.HasSuffix(p, "adb-claw-observe.png") {
		t.Errorf("default png path = %q", p)
	}
}

type recordingPullCommander struct {
	png        []byte
	calls      [][]string
	deviceFile string
}

func (m *recordingPullCommander) Shell(args ...string) (*adb.Result, error) {
	m.calls = append(m.calls, append([]string{"shell"}, args...))
	if len(args) >= 1 && args[0] == "screencap" {
		if len(args) < 3 {
			return &adb.Result{ExitCode: 1, Stderr: "missing path"}, nil
		}
		m.deviceFile = args[2]
		return &adb.Result{}, nil
	}
	if len(args) >= 1 && args[0] == "rm" {
		return &adb.Result{}, nil
	}
	return &adb.Result{}, nil
}

func (m *recordingPullCommander) ExecOut(args ...string) ([]byte, error) {
	m.calls = append(m.calls, append([]string{"exec-out"}, args...))
	return nil, fmt.Errorf("stream should not be used in pull mode")
}

func (m *recordingPullCommander) RawCommand(args ...string) (*adb.Result, error) {
	m.calls = append(m.calls, append([]string{"raw"}, args...))
	if len(args) >= 3 && args[0] == "pull" {
		if err := os.WriteFile(args[2], m.png, 0644); err != nil {
			return nil, err
		}
		return &adb.Result{}, nil
	}
	return nil, fmt.Errorf("unexpected raw %v", args)
}

func TestCaptureScreenshotPullWritesFileAndCleansDevice(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "pull.jpg")
	cmd := &recordingPullCommander{png: solidPNG(80, 120)}

	result, err := CaptureScreenshot(cmd, CaptureOptions{
		Format:  "jpeg",
		Path:    path,
		Mode:    CaptureModePull,
		Profile: true,
	})
	if err != nil {
		t.Fatalf("CaptureScreenshot: %v", err)
	}
	if result.Mode != string(CaptureModePull) {
		t.Errorf("mode = %q", result.Mode)
	}
	if result.Profile == nil || result.Profile.ADBCalls < 3 {
		t.Fatalf("expected at least 3 ADB calls (screencap/pull/rm), got %+v", result.Profile)
	}
	sawPull, sawRm := false, false
	for _, c := range cmd.calls {
		if len(c) >= 2 && c[0] == "raw" && c[1] == "pull" {
			sawPull = true
		}
		if len(c) >= 2 && c[0] == "shell" && c[1] == "rm" {
			sawRm = true
		}
	}
	if !sawPull || !sawRm {
		t.Fatalf("expected pull and rm, calls=%v", cmd.calls)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatal(err)
	}
}

func TestTakeScreenshotPNG(t *testing.T) {
	cmd := &mockCaptureCommander{png: solidPNG(64, 32)}
	data, err := TakeScreenshot(cmd, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(data) < 8 || string(data[1:4]) != "PNG" {
		t.Fatal("TakeScreenshot should return PNG bytes")
	}
}
