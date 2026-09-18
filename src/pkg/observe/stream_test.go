package observe

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/llm-net/adb-claw/pkg/adb"
	"github.com/llm-net/adb-claw/pkg/frameartifact"
	"github.com/llm-net/adb-claw/pkg/stream"
)

type failCaptureCommander struct{}

func (failCaptureCommander) Shell(args ...string) (*adb.Result, error) {
	return &adb.Result{}, nil
}
func (failCaptureCommander) ExecOut(args ...string) ([]byte, error) {
	return nil, fmt.Errorf("screencap should not run when a live frame exists")
}
func (failCaptureCommander) RawCommand(args ...string) (*adb.Result, error) {
	return nil, fmt.Errorf("raw should not run")
}

func colorJPEG(width, height int, c color.Color) []byte {
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.Set(x, y, c)
		}
	}
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 80}); err != nil {
		panic(err)
	}
	return buf.Bytes()
}

func writePumpFrame(t *testing.T, jpeg []byte, seq uint32, rotation int) *stream.Latest {
	t.Helper()
	key := stream.DefaultKey
	if err := os.MkdirAll(stream.Dir(key), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(stream.JPEGPath(key), jpeg, 0644); err != nil {
		t.Fatal(err)
	}
	latest := stream.Latest{
		Seq:           seq,
		Hash:          frameartifact.Hash(jpeg),
		CapturedAt:    time.Now().UTC(),
		UpdatedAt:     time.Now().UTC(),
		Path:          stream.JPEGPath(key),
		Source:        "dex",
		Mode:          "stream",
		DeviceWidth:   100,
		DeviceHeight:  200,
		ImageWidth:    100,
		ImageHeight:   200,
		Rotation:      rotation,
		RotationKnown: true,
		Complete:      true,
	}
	if err := stream.Write(key, latest); err != nil {
		t.Fatal(err)
	}
	return &latest
}

func TestObserveCopiesLatestStreamFrame(t *testing.T) {
	t.Setenv("TMPDIR", t.TempDir())
	jpeg := colorJPEG(100, 200, color.RGBA{R: 200, A: 255})
	writePumpFrame(t, jpeg, 4, 0)

	result := Observe(failCaptureCommander{}, ObserveOptions{Format: "jpeg"})
	if result.Screenshot == nil {
		t.Fatalf("errors=%v", result.Errors)
	}
	ss := result.Screenshot
	if ss.Source != "pump" || ss.FrameSeq != 4 {
		t.Fatalf("%+v", ss)
	}
	if ss.Path == stream.JPEGPath(stream.DefaultKey) {
		t.Fatal("observe must copy latest.jpg to a unique token path")
	}
	if _, err := os.Stat(ss.Path); err != nil {
		t.Fatal(err)
	}
	if ss.DeviceWidth != 100 || ss.ActionHeight != 200 {
		t.Fatalf("action space %+v", ss)
	}
	if ss.FrameToken == "" {
		t.Fatal("missing frame token")
	}
	if _, err := frameartifact.Load(ss.FrameToken); err != nil {
		t.Fatal(err)
	}
}

func TestWaitForChangeReadsPumpLatest(t *testing.T) {
	t.Setenv("TMPDIR", t.TempDir())
	firstJPEG := colorJPEG(100, 200, color.Black)
	first := writePumpFrame(t, firstJPEG, 1, 0)
	meta := &frameartifact.Metadata{
		Token:        "tok",
		Hash:         first.Hash,
		Format:       "jpeg",
		CaptureMode:  "stream",
		Quality:      60,
		DeviceWidth:  100,
		DeviceHeight: 200,
		ActionWidth:  100,
		ActionHeight: 200,
	}
	go func() {
		time.Sleep(25 * time.Millisecond)
		writePumpFrame(t, colorJPEG(100, 200, color.White), 2, 0)
	}()
	result, err := WaitForChange(failCaptureCommander{}, meta, time.Second, 5*time.Millisecond)
	if err != nil {
		t.Fatal(err)
	}
	if !result.Changed || result.Screenshot == nil {
		t.Fatalf("%+v", result)
	}
	if result.Screenshot.Path == stream.JPEGPath(stream.DefaultKey) {
		t.Fatal("changed frame must be a unique copy")
	}
	if result.Screenshot.Hash == first.Hash {
		t.Fatal("expected a new hash")
	}
}

func TestObservePullModeSkipsStream(t *testing.T) {
	t.Setenv("TMPDIR", t.TempDir())
	writePumpFrame(t, colorJPEG(40, 80, color.White), 9, 0)
	if useLiveStream(ObserveOptions{Format: "jpeg", Mode: CaptureModePull}) {
		t.Fatal("explicit pull must skip the livestream")
	}
	cmd := &mockObserveCommander{png: solidPNG(40, 80)}
	result := Observe(cmd, ObserveOptions{Format: "jpeg", DisableStream: true})
	if result.Screenshot == nil {
		t.Fatalf("%v", result.Errors)
	}
	if result.Screenshot.Source == "pump" {
		t.Fatal("disabled stream must recapture, not copy the livestream")
	}
}

func TestSnapshotNotLatestPath(t *testing.T) {
	t.Setenv("TMPDIR", t.TempDir())
	jpeg := colorJPEG(20, 40, color.RGBA{B: 255, A: 255})
	latest := writePumpFrame(t, jpeg, 1, 3)
	dest := filepath.Join(t.TempDir(), "copy.jpg")
	ss, err := snapshotLatest(nil, latest, ObserveOptions{Path: dest, Format: "jpeg"})
	if err != nil {
		t.Fatal(err)
	}
	if ss.Rotation != 3 || ss.Path != dest {
		t.Fatalf("%+v", ss)
	}
}
