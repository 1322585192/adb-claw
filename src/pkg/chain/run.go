package chain

import (
	"fmt"
	"time"

	"github.com/llm-net/adb-claw/pkg/adb"
	"github.com/llm-net/adb-claw/pkg/input"
)

// Run injects already-mapped device-pixel steps. gap is the gesture pause
// between steps (not applied after the last step) and must not observe.
func Run(cmd adb.Commander, steps []Step, gap time.Duration) error {
	if len(steps) == 0 {
		return fmt.Errorf("no steps")
	}
	for i, step := range steps {
		if err := runStep(cmd, step); err != nil {
			return StepError{Step: i + 1, Completed: i, Err: err}
		}
		if i < len(steps)-1 && gap > 0 {
			time.Sleep(gap)
		}
	}
	return nil
}

func runStep(cmd adb.Commander, step Step) error {
	switch step.Kind {
	case KindTap:
		return input.Tap(cmd, step.X, step.Y)
	case KindLongPress:
		return input.LongPress(cmd, step.X, step.Y, durationOr(step.DurationMs, DefaultLongPressMs))
	case KindSwipe:
		return input.Swipe(cmd, step.X, step.Y, step.X2, step.Y2, durationOr(step.DurationMs, DefaultSwipeMs))
	default:
		return fmt.Errorf("unknown chain action %q", step.Kind)
	}
}

func durationOr(value, fallback int) int {
	if value <= 0 {
		return fallback
	}
	return value
}
