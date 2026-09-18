package stream

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/llm-net/adb-claw/pkg/atomicfile"
	"github.com/llm-net/adb-claw/pkg/frameartifact"
)

// FreshAge is how recent latest.json must be before observe copies it
// without waiting for another pump tick.
const FreshAge = 2 * time.Second

// Latest is the capacity-1 sidecar written after each encoded JPEG.
type Latest struct {
	Seq           uint32    `json:"seq"`
	Hash          string    `json:"hash"`
	CapturedAt    time.Time `json:"captured_at"`
	UpdatedAt     time.Time `json:"updated_at"`
	Path          string    `json:"path"`
	Source        string    `json:"source,omitempty"`
	Mode          string    `json:"mode,omitempty"`
	DeviceWidth   int       `json:"device_width"`
	DeviceHeight  int       `json:"device_height"`
	ImageWidth    int       `json:"image_width"`
	ImageHeight   int       `json:"image_height"`
	Rotation      int       `json:"rotation"`
	RotationKnown bool      `json:"rotation_known"`
	Complete      bool      `json:"complete"`
	Quality       int       `json:"quality,omitempty"`
	Width         int       `json:"width,omitempty"`
	Serial        string    `json:"serial,omitempty"`
}

// Write atomically stores latest.json after latest.jpg has been replaced.
func Write(key string, latest Latest) error {
	if latest.UpdatedAt.IsZero() {
		latest.UpdatedAt = time.Now().UTC()
	}
	if latest.Path == "" {
		latest.Path = JPEGPath(key)
	}
	data, err := json.Marshal(latest)
	if err != nil {
		return err
	}
	return atomicfile.Write(JSONPath(key), data, 0644)
}

// Read loads the current sidecar. It does not verify the JPEG still matches.
func Read(key string) (*Latest, error) {
	data, err := os.ReadFile(JSONPath(key))
	if err != nil {
		return nil, err
	}
	var latest Latest
	if err := json.Unmarshal(data, &latest); err != nil {
		return nil, fmt.Errorf("decode latest frame: %w", err)
	}
	if latest.Path == "" {
		latest.Path = JPEGPath(key)
	}
	return &latest, nil
}

// ReadJPEG returns the current latest.jpg bytes. It retries when the file
// hash does not yet match the sidecar (atomic rename race).
func ReadJPEG(latest *Latest) ([]byte, error) {
	if latest == nil {
		return nil, fmt.Errorf("no latest frame")
	}
	var lastErr error
	for i := 0; i < 4; i++ {
		data, err := os.ReadFile(latest.Path)
		if err != nil {
			lastErr = err
			time.Sleep(15 * time.Millisecond)
			continue
		}
		if len(data) == 0 {
			lastErr = fmt.Errorf("empty latest jpeg")
			time.Sleep(15 * time.Millisecond)
			continue
		}
		hash := frameartifact.Hash(data)
		if latest.Hash == "" || hash == latest.Hash {
			return data, nil
		}
		lastErr = fmt.Errorf("latest jpeg hash mismatch")
		time.Sleep(15 * time.Millisecond)
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("latest jpeg unavailable")
	}
	return nil, lastErr
}

// Age is time since the sidecar was written (UpdatedAt, else CapturedAt).
func Age(latest *Latest) time.Duration {
	if latest == nil {
		return 0
	}
	if !latest.UpdatedAt.IsZero() {
		return time.Since(latest.UpdatedAt)
	}
	if !latest.CapturedAt.IsZero() {
		return time.Since(latest.CapturedAt)
	}
	return 0
}

// Fresh reports whether a sidecar exists and was updated within maxAge.
func Fresh(key string, maxAge time.Duration) bool {
	latest, err := Read(key)
	if err != nil {
		return false
	}
	if maxAge <= 0 {
		maxAge = FreshAge
	}
	return Age(latest) <= maxAge
}

// Wait polls latest.json until pred returns true or timeout expires.
func Wait(key string, timeout time.Duration, pred func(*Latest) bool) (*Latest, error) {
	if timeout <= 0 {
		timeout = 4 * time.Second
	}
	if pred == nil {
		pred = func(latest *Latest) bool { return latest != nil && latest.Hash != "" }
	}
	deadline := time.Now().Add(timeout)
	var lastErr error
	for {
		latest, err := Read(key)
		if err == nil && pred(latest) {
			return latest, nil
		}
		if err != nil {
			lastErr = err
		} else {
			lastErr = fmt.Errorf("latest frame not ready")
		}
		if !time.Now().Before(deadline) {
			if lastErr == nil {
				lastErr = fmt.Errorf("latest frame timeout")
			}
			return latest, lastErr
		}
		time.Sleep(20 * time.Millisecond)
	}
}
