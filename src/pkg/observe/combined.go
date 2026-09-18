package observe

import (
	"github.com/llm-net/adb-claw/pkg/adb"
	"github.com/llm-net/adb-claw/pkg/perf"
)

// ObserveResult is a screenshot-only observation. JSON never includes image bytes.
type ObserveResult struct {
	Screenshot *ScreenshotResult `json:"screenshot,omitempty"`
	Errors     []string          `json:"errors,omitempty"`
	Profile    *ObserveProfile   `json:"profile,omitempty"`
}

// Observe captures the current screen image. There is no UI tree.
func Observe(cmd adb.Commander, opts ObserveOptions) *ObserveResult {
	clock := perf.Start()
	result := &ObserveResult{}

	format := formatOrJPEG(opts.Format)
	path := opts.Path
	if path == "" {
		path = DefaultObservePath(format)
	}
	ss, err := CaptureScreenshot(cmd, CaptureOptions{
		MaxWidth:  opts.MaxWidth,
		MaxPixels: opts.MaxPixels,
		Format:    format,
		Quality:   opts.Quality,
		Path:      path,
		Mode:      opts.Mode,
		Profile:   opts.Profile,
	})
	if err != nil {
		result.Errors = append(result.Errors, "screenshot: "+err.Error())
	} else {
		result.Screenshot = ss
	}
	if opts.Profile {
		prof := &ObserveProfile{TotalMs: clock.Total()}
		if result.Screenshot != nil {
			prof.Screenshot = result.Screenshot.Profile
		}
		result.Profile = prof
	}
	if len(result.Errors) == 0 {
		result.Errors = nil
	}
	return result
}
