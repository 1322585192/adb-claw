package observe

import (
	"sync"

	"github.com/llm-net/adb-claw/pkg/adb"
	"github.com/llm-net/adb-claw/pkg/perf"
)

// ObserveResult holds the combined output of screenshot + UI tree.
type ObserveResult struct {
	StateID    string            `json:"state_id,omitempty"`
	Screenshot *ScreenshotResult `json:"screenshot,omitempty"`
	UI         *UITree           `json:"ui,omitempty"`
	Errors     []string          `json:"errors,omitempty"`
	Profile    *ObserveProfile   `json:"profile,omitempty"`
}

// ObserveEvent is a partial observe result that can be streamed as soon as
// screenshot or UI tree completes.
type ObserveEvent struct {
	Kind       string
	Screenshot *ScreenshotResult
	UI         *UITree
	Err        error
}

// Observe captures both screenshot and UI tree in parallel.
// Partial failure is tolerated — one failing doesn't block the other.
// Screenshot encoding follows opts; UI tree coordinates stay in device pixels.
// A screenshot file is always written (opts.Path or $TMPDIR/adb-claw-observe.jpg)
// unless SkipImage is set.
func Observe(cmd adb.Commander, opts ObserveOptions) *ObserveResult {
	clock := perf.Start()
	result := &ObserveResult{}
	var mu sync.Mutex
	var wg sync.WaitGroup

	if !opts.SkipImage {
		wg.Add(1)
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
				Mode:     opts.Mode,
				Profile:  opts.Profile,
			})
			mu.Lock()
			defer mu.Unlock()
			if err != nil {
				result.Errors = append(result.Errors, "screenshot: "+err.Error())
			} else {
				result.Screenshot = ss
			}
		}()
	}

	wg.Add(1)
	go func() {
		defer wg.Done()
		tree, err := DumpUITreeOpts(cmd, DumpOptions{
			Compressed: opts.Compressed,
			Mode:       opts.UIMode,
			Profile:    opts.Profile,
			Save:       true,
		})
		mu.Lock()
		defer mu.Unlock()
		if err != nil {
			result.Errors = append(result.Errors, "uitree: "+err.Error())
		} else {
			result.UI = tree
			result.StateID = tree.StateID
		}
	}()

	wg.Wait()

	if len(result.Errors) == 0 {
		result.Errors = nil
	}
	if opts.Profile {
		prof := &ObserveProfile{TotalMs: clock.Total()}
		if result.Screenshot != nil {
			prof.Screenshot = result.Screenshot.Profile
		}
		if result.UI != nil {
			prof.UI = result.UI.Profile
		}
		result.Profile = prof
	}
	return result
}

// ObserveStream runs screenshot and UI dump in parallel and invokes onEvent
// as soon as each part completes. The returned result is the aggregated view.
func ObserveStream(cmd adb.Commander, opts ObserveOptions, onEvent func(ObserveEvent)) *ObserveResult {
	if onEvent == nil {
		return Observe(cmd, opts)
	}

	clock := perf.Start()
	result := &ObserveResult{}
	var mu sync.Mutex
	var wg sync.WaitGroup

	if !opts.SkipImage {
		wg.Add(1)
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
				Mode:     opts.Mode,
				Profile:  opts.Profile,
			})
			mu.Lock()
			if err != nil {
				result.Errors = append(result.Errors, "screenshot: "+err.Error())
			} else {
				result.Screenshot = ss
			}
			mu.Unlock()
			onEvent(ObserveEvent{Kind: "screenshot", Screenshot: ss, Err: err})
		}()
	}

	wg.Add(1)
	go func() {
		defer wg.Done()
		tree, err := DumpUITreeOpts(cmd, DumpOptions{
			Compressed: opts.Compressed,
			Mode:       opts.UIMode,
			Profile:    opts.Profile,
			Save:       true,
		})
		mu.Lock()
		if err != nil {
			result.Errors = append(result.Errors, "uitree: "+err.Error())
		} else {
			result.UI = tree
			result.StateID = tree.StateID
		}
		mu.Unlock()
		onEvent(ObserveEvent{Kind: "ui", UI: tree, Err: err})
	}()

	wg.Wait()
	if len(result.Errors) == 0 {
		result.Errors = nil
	}
	if opts.Profile {
		prof := &ObserveProfile{TotalMs: clock.Total()}
		if result.Screenshot != nil {
			prof.Screenshot = result.Screenshot.Profile
		}
		if result.UI != nil {
			prof.UI = result.UI.Profile
		}
		result.Profile = prof
	}
	return result
}
