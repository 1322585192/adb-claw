package stream

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/llm-net/adb-claw/pkg/frameartifact"
)

func TestSanitizeSerial(t *testing.T) {
	if got := Sanitize("ABC123"); got != "ABC123" {
		t.Fatalf("got %q", got)
	}
	if got := Sanitize("192.168.1.8:5555"); got != "192_168_1_8_5555" {
		t.Fatalf("got %q", got)
	}
	if got := Sanitize("  "); got != DefaultKey {
		t.Fatalf("empty = %q", got)
	}
}

func TestWriteReadLatestMatchesJPEG(t *testing.T) {
	t.Setenv("TMPDIR", t.TempDir())
	jpeg := []byte{0xFF, 0xD8, 0xFF, 0x00, 0x01, 0x02}
	key := "dev1"
	if err := os.MkdirAll(Dir(key), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(JPEGPath(key), jpeg, 0644); err != nil {
		t.Fatal(err)
	}
	latest := Latest{
		Seq:          7,
		Hash:         frameartifact.Hash(jpeg),
		CapturedAt:   time.Now().UTC(),
		DeviceWidth:  100,
		DeviceHeight: 200,
		ImageWidth:   50,
		ImageHeight:  100,
		Rotation:     1,
		Complete:     true,
		Source:       "dex",
		Mode:         "stream",
	}
	if err := Write(key, latest); err != nil {
		t.Fatal(err)
	}
	got, err := Read(key)
	if err != nil {
		t.Fatal(err)
	}
	if got.Seq != 7 || got.Rotation != 1 || got.Path != JPEGPath(key) {
		t.Fatalf("%+v", got)
	}
	data, err := ReadJPEG(got)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != string(jpeg) {
		t.Fatal("jpeg mismatch")
	}
	if !Fresh(key, time.Second) {
		t.Fatal("expected fresh sidecar")
	}
}

func TestWaitSeesUpdatedHash(t *testing.T) {
	t.Setenv("TMPDIR", t.TempDir())
	key := DefaultKey
	if err := os.MkdirAll(Dir(key), 0755); err != nil {
		t.Fatal(err)
	}
	first := Latest{Seq: 1, Hash: "aaa", Path: JPEGPath(key)}
	if err := Write(key, first); err != nil {
		t.Fatal(err)
	}
	go func() {
		time.Sleep(30 * time.Millisecond)
		_ = Write(key, Latest{Seq: 2, Hash: "bbb", Path: JPEGPath(key)})
	}()
	got, err := Wait(key, time.Second, func(latest *Latest) bool {
		return latest != nil && latest.Hash == "bbb"
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.Seq != 2 {
		t.Fatalf("%+v", got)
	}
}

func TestRunningFalseWithoutPID(t *testing.T) {
	t.Setenv("TMPDIR", t.TempDir())
	if Running(DefaultKey) {
		t.Fatal("expected no pump")
	}
	if err := os.MkdirAll(Dir(DefaultKey), 0755); err != nil {
		t.Fatal(err)
	}
	if err := WritePID(DefaultKey, os.Getpid()); err != nil {
		t.Fatal(err)
	}
	if !Running(DefaultKey) {
		t.Fatal("current test process should count as alive")
	}
	if filepath.Base(JPEGPath(DefaultKey)) != "latest.jpg" {
		t.Fatal(JPEGPath(DefaultKey))
	}
}
