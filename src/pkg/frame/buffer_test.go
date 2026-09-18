package frame

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestBufferKeepsOnlyLatest(t *testing.T) {
	b := NewBuffer()
	b.Put(&Frame{Header: Header{Seq: 1}, JPEG: []byte("a"), Hash: "a"})
	b.Put(&Frame{Header: Header{Seq: 2}, JPEG: []byte("b"), Hash: "b"})
	b.Put(&Frame{Header: Header{Seq: 3}, JPEG: []byte("c"), Hash: "c"})
	got := b.Latest()
	if got == nil || got.Seq != 3 {
		t.Fatalf("latest = %+v", got)
	}
	if b.Dropped() != 2 {
		t.Fatalf("dropped = %d", b.Dropped())
	}
}

func TestBufferMarksDuplicates(t *testing.T) {
	b := NewBuffer()
	b.Put(&Frame{Header: Header{Seq: 1}, JPEG: []byte("same"), Hash: "h1"})
	b.Put(&Frame{Header: Header{Seq: 2}, JPEG: []byte("same"), Hash: "h1"})
	if !b.Latest().Duplicate {
		t.Fatal("expected duplicate mark")
	}
}

func TestWaitAfterHashChange(t *testing.T) {
	b := NewBuffer()
	b.Put(&Frame{Header: Header{Seq: 1}, JPEG: []byte("a"), Hash: "a"})
	done := make(chan *Frame, 1)
	go func() {
		done <- b.WaitAfter(1, "a", time.Second)
	}()
	time.Sleep(20 * time.Millisecond)
	b.Put(&Frame{Header: Header{Seq: 2}, JPEG: []byte("b"), Hash: "b"})
	got := <-done
	if got == nil || got.Seq != 2 {
		t.Fatalf("waited frame %+v", got)
	}
}

func TestWaitAfterTimeoutReturnsLatest(t *testing.T) {
	b := NewBuffer()
	b.Put(&Frame{Header: Header{Seq: 4}, JPEG: []byte("a"), Hash: "a"})
	got := b.WaitAfter(4, "a", 20*time.Millisecond)
	if got == nil || got.Seq != 4 {
		t.Fatalf("timeout should return current latest, got %+v", got)
	}
}

func TestWriteAtomicAndCleanup(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "latest.jpg")
	if err := WriteAtomic(path, []byte("jpeg-bytes")); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "jpeg-bytes" {
		t.Fatalf("got %q", data)
	}
	if _, err := os.Stat(path + ".tmp"); !os.IsNotExist(err) {
		t.Fatal("temp file should be gone")
	}
}
