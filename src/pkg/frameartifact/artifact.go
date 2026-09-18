package frameartifact

import (
	"crypto/rand"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/llm-net/adb-claw/pkg/apptmp"
	"github.com/llm-net/adb-claw/pkg/atomicfile"
)

const (
	// MaxFrames bounds temporary one-shot frame artifacts.
	MaxFrames = 32
	// MaxAge removes abandoned temporary frame metadata.
	MaxAge = 10 * time.Minute
)

// Metadata binds an image artifact to the coordinate space used to act on it.
type Metadata struct {
	Token         string    `json:"frame_token"`
	Hash          string    `json:"hash"`
	CapturedAt    time.Time `json:"captured_at"`
	Path          string    `json:"path"`
	Format        string    `json:"format"`
	CaptureMode   string    `json:"capture_mode"`
	Quality       int       `json:"quality"`
	MaxWidth      int       `json:"max_width,omitempty"`
	MaxPixels     int       `json:"max_pixels,omitempty"`
	DeviceWidth   int       `json:"device_width"`
	DeviceHeight  int       `json:"device_height"`
	ActionWidth   int       `json:"action_width"`
	ActionHeight  int       `json:"action_height"`
	ImageWidth    int       `json:"image_width"`
	ImageHeight   int       `json:"image_height"`
	Rotation      int       `json:"rotation"`
	RotationKnown bool      `json:"rotation_known"`
	Complete      bool      `json:"complete"`
}

// Dir is the private temporary directory for one-shot frames and metadata.
func Dir() string {
	return filepath.Join(apptmp.Root(), "adb-claw", "frames")
}

// NewToken returns an opaque, filename-safe frame identifier.
func NewToken() string {
	var suffix [4]byte
	if _, err := rand.Read(suffix[:]); err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return fmt.Sprintf("%d-%s", time.Now().UnixNano(), hex.EncodeToString(suffix[:]))
}

// Path returns the default unique image path for token.
func Path(token, format string) string {
	ext := "jpg"
	if strings.EqualFold(format, "png") {
		ext = "png"
	}
	return filepath.Join(Dir(), token+"."+ext)
}

// TokenFromPath extracts the token from a default frame artifact path.
func TokenFromPath(path string) (string, bool) {
	if !insideDir(path, Dir()) {
		return "", false
	}
	token := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	return token, validToken(token)
}

// Hash returns a compact content fingerprint.
func Hash(data []byte) string {
	sum := sha1.Sum(data)
	return hex.EncodeToString(sum[:8])
}

// Save atomically stores metadata and prunes old temporary frame artifacts.
func Save(meta Metadata) error {
	if !validToken(meta.Token) {
		return fmt.Errorf("invalid frame token")
	}
	data, err := json.Marshal(meta)
	if err != nil {
		return err
	}
	if err := atomicfile.Write(metadataPath(meta.Token), data, 0644); err != nil {
		return err
	}
	prune(time.Now())
	return nil
}

// Load resolves a token created by a previous CLI process.
func Load(token string) (*Metadata, error) {
	if !validToken(token) {
		return nil, fmt.Errorf("invalid frame token")
	}
	data, err := os.ReadFile(metadataPath(token))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("unknown or expired frame token %q", token)
		}
		return nil, err
	}
	var meta Metadata
	if err := json.Unmarshal(data, &meta); err != nil {
		return nil, fmt.Errorf("decode frame metadata: %w", err)
	}
	if meta.Token != token || meta.ActionWidth <= 0 || meta.ActionHeight <= 0 {
		return nil, fmt.Errorf("invalid metadata for frame token %q", token)
	}
	return &meta, nil
}

func metadataPath(token string) string {
	return filepath.Join(Dir(), token+".json")
}

func validToken(token string) bool {
	if token == "" || len(token) > 96 {
		return false
	}
	for _, r := range token {
		if (r < 'a' || r > 'z') && (r < 'A' || r > 'Z') &&
			(r < '0' || r > '9') && r != '-' && r != '_' {
			return false
		}
	}
	return true
}

type entry struct {
	token string
	meta  Metadata
	mod   time.Time
}

func prune(now time.Time) {
	files, err := os.ReadDir(Dir())
	if err != nil {
		return
	}
	entries := make([]entry, 0, len(files))
	for _, file := range files {
		if file.IsDir() || filepath.Ext(file.Name()) != ".json" {
			continue
		}
		token := strings.TrimSuffix(file.Name(), ".json")
		data, err := os.ReadFile(filepath.Join(Dir(), file.Name()))
		if err != nil {
			continue
		}
		var meta Metadata
		if json.Unmarshal(data, &meta) != nil || meta.Token != token {
			continue
		}
		info, err := file.Info()
		if err != nil {
			continue
		}
		entries = append(entries, entry{token: token, meta: meta, mod: info.ModTime()})
	}
	sort.Slice(entries, func(i, j int) bool {
		if !entries[i].meta.CapturedAt.Equal(entries[j].meta.CapturedAt) {
			return entries[i].meta.CapturedAt.After(entries[j].meta.CapturedAt)
		}
		return entries[i].token > entries[j].token
	})
	for i, item := range entries {
		if i < MaxFrames && now.Sub(item.mod) <= MaxAge {
			continue
		}
		_ = os.Remove(metadataPath(item.token))
		if insideDir(item.meta.Path, Dir()) {
			_ = os.Remove(item.meta.Path)
		}
	}
}

func insideDir(path, dir string) bool {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return false
	}
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return false
	}
	rel, err := filepath.Rel(absDir, absPath)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}
