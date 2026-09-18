package observe

import (
	"os"
	"testing"
	"time"
)

func TestSaveAndLoadSnapshot(t *testing.T) {
	t.Setenv("TMPDIR", t.TempDir())
	tree, err := ParseUITree([]byte(sampleXML))
	if err != nil {
		t.Fatal(err)
	}
	snap, err := SaveSnapshot("dev1", tree)
	if err != nil {
		t.Fatal(err)
	}
	if snap.StateID == "" {
		t.Fatal("expected state_id")
	}

	got, err := LoadSnapshot("dev1", time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if got.StateID != snap.StateID {
		t.Errorf("state_id = %s, want %s", got.StateID, snap.StateID)
	}
	if len(got.Tree.Elements) != len(tree.Elements) {
		t.Errorf("elements = %d, want %d", len(got.Tree.Elements), len(tree.Elements))
	}

	if _, err := LoadSnapshotByID("dev1", "nope", time.Minute); err == nil {
		t.Error("expected stale state_id error")
	}
	if _, err := LoadSnapshotByID("dev1", snap.StateID, time.Minute); err != nil {
		t.Fatal(err)
	}
}

func TestLoadSnapshotExpired(t *testing.T) {
	t.Setenv("TMPDIR", t.TempDir())
	tree, _ := ParseUITree([]byte(sampleXML))
	if _, err := SaveSnapshot("dev2", tree); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadSnapshot("dev2", time.Nanosecond); err == nil {
		t.Fatal("expected expiry error")
	}
}

func TestLoadSnapshotMissing(t *testing.T) {
	t.Setenv("TMPDIR", t.TempDir())
	if _, err := LoadSnapshot("missing", time.Minute); err == nil {
		t.Fatal("expected missing snapshot error")
	}
}

func TestSnapshotPathIsolated(t *testing.T) {
	a := SnapshotPath("localhost:51427")
	b := SnapshotPath("emulator-5554")
	if a == b {
		t.Fatalf("expected different snapshot paths, both %s", a)
	}
	if _, err := os.Stat(a); err == nil {
		t.Fatalf("path should not exist yet: %s", a)
	}
}
