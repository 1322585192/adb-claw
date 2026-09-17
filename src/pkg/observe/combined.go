package observe

import (
	"sync"

	"github.com/llm-net/adb-claw/pkg/adb"
)

// ObserveResult holds the combined output of screenshot + UI tree.
type ObserveResult struct {
	Screenshot *ScreenshotResult `json:"screenshot,omitempty"`
	UI         *UITree           `json:"ui,omitempty"`
	Errors     []string          `json:"errors,omitempty"`
}

// Observe captures both screenshot and UI tree in parallel.
// Partial failure is tolerated — one failing doesn't block the other.
// Screenshot encoding follows opts; UI tree coordinates stay in device pixels.
// A screenshot file is always written (opts.Path or $TMPDIR/adb-claw-observe.jpg).
func Observe(cmd adb.Commander, opts ObserveOptions) *ObserveResult {
	result := &ObserveResult{}
	var mu sync.Mutex
	var wg sync.WaitGroup

	wg.Add(2)

	// Screenshot goroutine
	go func() {
		defer wg.Done()
		format := formatOrJPEG(opts.Format)
		path := opts.Path
		if path == "" {
			path = DefaultObservePath(format)
		}
		ss, err := CaptureScreenshot(cmd, CaptureOptions{
			MaxWidth: opts.MaxWidth,
			Format:   format,
			Quality:  opts.Quality,
			Path:     path,
		})
		mu.Lock()
		defer mu.Unlock()
		if err != nil {
			result.Errors = append(result.Errors, "screenshot: "+err.Error())
		} else {
			result.Screenshot = ss
		}
	}()

	// UI tree goroutine
	go func() {
		defer wg.Done()
		tree, err := DumpUITree(cmd)
		mu.Lock()
		defer mu.Unlock()
		if err != nil {
			result.Errors = append(result.Errors, "uitree: "+err.Error())
		} else {
			result.UI = tree
		}
	}()

	wg.Wait()

	// nil out empty errors slice for cleaner JSON
	if len(result.Errors) == 0 {
		result.Errors = nil
	}

	return result
}
