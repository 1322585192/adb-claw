package observe

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"path/filepath"
	"testing"
	"time"

	"github.com/llm-net/adb-claw/pkg/frameartifact"
)

func colorPNG(width, height int, c color.Color) []byte {
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.Set(x, y, c)
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		panic(err)
	}
	return buf.Bytes()
}

func TestWaitForChangeUsesPriorFrameHash(t *testing.T) {
	t.Setenv("TMPDIR", t.TempDir())
	base := colorPNG(100, 200, color.Black)
	changed := colorPNG(100, 200, color.White)
	cmd := &retryCaptureCommander{
		size: "Physical size: 100x200\n",
		pngs: [][]byte{base, base, changed},
	}
	first, err := CaptureScreenshot(cmd, CaptureOptions{Format: "jpeg"})
	if err != nil {
		t.Fatal(err)
	}
	meta, err := frameartifact.Load(first.FrameToken)
	if err != nil {
		t.Fatal(err)
	}
	result, err := WaitForChange(cmd, meta, time.Second, time.Millisecond)
	if err != nil {
		t.Fatal(err)
	}
	if !result.Changed || result.Attempts != 2 {
		t.Fatalf("result = %+v", result)
	}
	if result.Screenshot.FrameToken == first.FrameToken || result.Screenshot.Path == first.Path {
		t.Fatal("changed frame must have a new token and unique path")
	}
	jpgs, err := filepath.Glob(filepath.Join(frameartifact.Dir(), "*.jpg"))
	if err != nil {
		t.Fatal(err)
	}
	if len(jpgs) != 2 {
		t.Fatalf("persisted jpegs = %d, want 2 (baseline + final wait frame)", len(jpgs))
	}
}

func TestWaitForChangeTimeoutReturnsLatestFrame(t *testing.T) {
	t.Setenv("TMPDIR", t.TempDir())
	base := solidPNG(40, 80)
	cmd := &retryCaptureCommander{
		size: "Physical size: 40x80\n",
		pngs: [][]byte{base, base},
	}
	first, err := CaptureScreenshot(cmd, CaptureOptions{Format: "jpeg"})
	if err != nil {
		t.Fatal(err)
	}
	meta, err := frameartifact.Load(first.FrameToken)
	if err != nil {
		t.Fatal(err)
	}
	result, err := WaitForChange(cmd, meta, time.Nanosecond, time.Millisecond)
	if err != nil {
		t.Fatal(err)
	}
	if result.Changed || result.Screenshot == nil {
		t.Fatalf("result = %+v", result)
	}
}
