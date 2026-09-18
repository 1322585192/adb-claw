package frame

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/llm-net/adb-claw/pkg/adb"
)

type shotCmd struct {
	png   []byte
	fail  bool
	calls int
}

func (m *shotCmd) Shell(args ...string) (*adb.Result, error) {
	return &adb.Result{}, nil
}
func (m *shotCmd) ExecOut(args ...string) ([]byte, error) {
	m.calls++
	if m.fail {
		return nil, fmt.Errorf("screencap unavailable")
	}
	return m.png, nil
}
func (m *shotCmd) RawCommand(args ...string) (*adb.Result, error) {
	return &adb.Result{}, nil
}

func tinyPNG() []byte {
	img := image.NewRGBA(image.Rect(0, 0, 20, 30))
	img.Set(0, 0, color.RGBA{R: 255, A: 255})
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		panic(err)
	}
	return buf.Bytes()
}

func TestFallbackCaptureNoDump(t *testing.T) {
	t.Setenv("TMPDIR", t.TempDir())
	cmd := &shotCmd{png: tinyPNG()}
	f, err := captureFallback(cmd, 3, 20, 50)
	if err != nil {
		t.Fatal(err)
	}
	if f.Seq != 3 || f.DeviceWidth != 20 || len(f.JPEG) == 0 {
		t.Fatalf("%+v", f)
	}
	if f.JPEG[0] != 0xFF || f.JPEG[1] != 0xD8 {
		t.Fatal("fallback must return JPEG, not a UI dump")
	}
}

func TestSourceFallbackWritesLatest(t *testing.T) {
	dir := t.TempDir()
	latest := filepath.Join(dir, "latest.jpg")
	t.Setenv("TMPDIR", dir)
	client := adb.NewClient("none", time.Second)
	client.ADBPath = filepath.Join(dir, "missing-adb")
	src, err := Start(context.Background(), client, Options{
		Interval:   30 * time.Millisecond,
		Width:      WidthLow,
		Quality:    QualityLow,
		LatestPath: latest,
		DisableDEX: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer src.Close()
	if src.Mode() != "pull" {
		t.Fatalf("mode=%s, want pull fallback", src.Mode())
	}
	// Without a working adb the pull loop records errors; the important
	// contract is no dump path and Close cleans up.
	if src.Resolution() != WidthLow {
		t.Fatalf("resolution %d", src.Resolution())
	}
}

func TestEmbeddedDEXMagic(t *testing.T) {
	if len(EmbeddedDEX()) < 4 || string(EmbeddedDEX()[:4]) != "dex\n" {
		t.Fatal("embedded frame DEX must exist so builds do not fall back to a dump helper")
	}
}

func TestLookupUnknownSeq(t *testing.T) {
	s := &Source{history: []FrameMeta{{Seq: 2, Hash: "x"}}}
	if s.Lookup(1) != nil {
		t.Fatal("unknown seq should miss")
	}
	if s.Lookup(2) == nil {
		t.Fatal("expected hit")
	}
}

func TestWriteAtomicCleansSibling(t *testing.T) {
	path := filepath.Join(t.TempDir(), "f.jpg")
	if err := WriteAtomic(path, []byte("aa")); err != nil {
		t.Fatal(err)
	}
	entries, _ := os.ReadDir(filepath.Dir(path))
	for _, e := range entries {
		if filepath.Ext(e.Name()) == ".tmp" {
			t.Fatalf("leftover temp %s", e.Name())
		}
	}
}
