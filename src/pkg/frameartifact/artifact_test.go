package frameartifact

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestSaveLoadAndPrune(t *testing.T) {
	t.Setenv("TMPDIR", t.TempDir())
	var newest string
	for i := 0; i < MaxFrames+3; i++ {
		token := NewToken()
		path := Path(token, "jpeg")
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(token), 0644); err != nil {
			t.Fatal(err)
		}
		meta := Metadata{
			Token:        token,
			Hash:         Hash([]byte(token)),
			CapturedAt:   time.Now().UTC(),
			Path:         path,
			Format:       "jpeg",
			DeviceWidth:  1080,
			DeviceHeight: 2340,
			ActionWidth:  1080,
			ActionHeight: 2340,
			ImageWidth:   1080,
			ImageHeight:  2340,
			Complete:     true,
		}
		if err := Save(meta); err != nil {
			t.Fatal(err)
		}
		newest = token
	}

	got, err := Load(newest)
	if err != nil {
		t.Fatal(err)
	}
	if got.Token != newest || got.ActionHeight != 2340 {
		t.Fatalf("metadata = %+v", got)
	}
	metas, err := filepath.Glob(filepath.Join(Dir(), "*.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(metas) > MaxFrames {
		t.Fatalf("metadata count = %d, want <= %d", len(metas), MaxFrames)
	}
}

func TestPruneRemovesOrphanImages(t *testing.T) {
	t.Setenv("TMPDIR", t.TempDir())
	token := NewToken()
	path := Path(token, "jpeg")
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("orphan"), 0644); err != nil {
		t.Fatal(err)
	}
	Prune()
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("orphan jpeg still present: %v", err)
	}
}

func TestLoadRejectsTraversal(t *testing.T) {
	t.Setenv("TMPDIR", t.TempDir())
	if _, err := Load("../secret"); err == nil {
		t.Fatal("expected invalid token error")
	}
}
