package observe

import (
	"fmt"
	"time"

	"github.com/llm-net/adb-claw/pkg/adb"
	"github.com/llm-net/adb-claw/pkg/frameartifact"
)

// ChangeResult is the newest captured frame and whether it differs from the
// frame token used as the baseline.
type ChangeResult struct {
	Changed    bool              `json:"changed"`
	Attempts   int               `json:"attempts"`
	Screenshot *ScreenshotResult `json:"screenshot"`
}

// WaitForChange polls screenshots until content, rotation, or action-space
// dimensions differ from baseline. On timeout it returns the latest frame.
func WaitForChange(cmd adb.Commander, baseline *frameartifact.Metadata, timeout, interval time.Duration) (*ChangeResult, error) {
	if baseline == nil {
		return nil, fmt.Errorf("baseline frame metadata is required")
	}
	if timeout <= 0 {
		timeout = 1500 * time.Millisecond
	}
	if interval <= 0 {
		interval = 100 * time.Millisecond
	}
	deadline := time.Now().Add(timeout)
	result := &ChangeResult{}
	for {
		result.Attempts++
		frame, err := CaptureScreenshot(cmd, CaptureOptions{
			MaxWidth:  baseline.MaxWidth,
			MaxPixels: baseline.MaxPixels,
			Format:    baseline.Format,
			Quality:   baseline.Quality,
			Mode:      CaptureMode(baseline.CaptureMode),
		})
		if err != nil {
			return nil, err
		}
		result.Screenshot = frame
		result.Changed = frame.Hash != baseline.Hash ||
			frame.Rotation != baseline.Rotation ||
			frame.ActionWidth != baseline.ActionWidth ||
			frame.ActionHeight != baseline.ActionHeight
		if result.Changed || !time.Now().Before(deadline) {
			return result, nil
		}
		time.Sleep(minDuration(interval, time.Until(deadline)))
	}
}

func minDuration(a, b time.Duration) time.Duration {
	if a < b {
		return a
	}
	if b < 0 {
		return 0
	}
	return b
}
