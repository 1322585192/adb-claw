package atomicfile

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWriteReplacesCompleteFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "frame.jpg")
	if err := Write(path, []byte("old"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := Write(path, []byte("new-complete"), 0644); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "new-complete" {
		t.Fatalf("data = %q", data)
	}
	matches, err := filepath.Glob(filepath.Join(filepath.Dir(path), ".*.tmp-*"))
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 0 {
		t.Fatalf("temporary files left behind: %v", matches)
	}
}
