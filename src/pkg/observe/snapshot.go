package observe

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// DefaultSnapshotMaxAge is how long a cached UI snapshot remains usable for act.
const DefaultSnapshotMaxAge = 30 * time.Second

// Snapshot is a versioned UI tree that actions can target without re-dumping.
type Snapshot struct {
	StateID    string    `json:"state_id"`
	Serial     string    `json:"serial,omitempty"`
	CapturedAt time.Time `json:"captured_at"`
	Tree       *UITree   `json:"tree"`
}

// SnapshotPath returns the on-disk cache path for a device serial.
func SnapshotPath(serial string) string {
	name := "adb-claw-last-state"
	if serial != "" {
		sum := sha1.Sum([]byte(serial))
		name += "-" + hex.EncodeToString(sum[:6])
	}
	return filepath.Join(os.TempDir(), name+".json")
}

// NewStateID returns a short unique id for a freshly captured tree.
func NewStateID() string {
	sum := sha1.Sum([]byte(fmt.Sprintf("%d", time.Now().UnixNano())))
	return hex.EncodeToString(sum[:8])
}

// SaveSnapshot writes the latest UI tree for serial and returns the snapshot.
func SaveSnapshot(serial string, tree *UITree) (*Snapshot, error) {
	if tree == nil {
		return nil, fmt.Errorf("no UI tree to save")
	}
	if tree.StateID == "" {
		tree.StateID = NewStateID()
	}
	snap := &Snapshot{
		StateID:    tree.StateID,
		Serial:     serial,
		CapturedAt: time.Now().UTC(),
		Tree:       tree,
	}
	data, err := json.Marshal(snap)
	if err != nil {
		return nil, fmt.Errorf("marshal snapshot: %w", err)
	}
	if err := os.WriteFile(SnapshotPath(serial), data, 0644); err != nil {
		return nil, fmt.Errorf("write snapshot: %w", err)
	}
	return snap, nil
}

// LoadSnapshot reads the last saved tree. It fails if missing or older than maxAge.
func LoadSnapshot(serial string, maxAge time.Duration) (*Snapshot, error) {
	data, err := os.ReadFile(SnapshotPath(serial))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("no cached UI snapshot; run observe first, pass coordinates, or use --refresh")
		}
		return nil, fmt.Errorf("read snapshot: %w", err)
	}
	var snap Snapshot
	if err := json.Unmarshal(data, &snap); err != nil {
		return nil, fmt.Errorf("parse snapshot: %w", err)
	}
	if snap.Tree == nil {
		return nil, fmt.Errorf("cached UI snapshot is empty; run observe again")
	}
	if maxAge > 0 && time.Since(snap.CapturedAt) > maxAge {
		return nil, fmt.Errorf("cached UI snapshot %s expired after %s; run observe again or pass --refresh", snap.StateID, maxAge)
	}
	return &snap, nil
}

// LoadSnapshotByID loads the last snapshot only if its state_id matches.
func LoadSnapshotByID(serial, stateID string, maxAge time.Duration) (*Snapshot, error) {
	snap, err := LoadSnapshot(serial, maxAge)
	if err != nil {
		return nil, err
	}
	if stateID != "" && snap.StateID != stateID {
		return nil, fmt.Errorf("stale state_id %s (latest is %s); re-observe before acting", stateID, snap.StateID)
	}
	return snap, nil
}
