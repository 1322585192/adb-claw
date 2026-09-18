package chain

import "fmt"

const (
	MinSteps           = 2
	MaxSteps           = 8
	DefaultGapMs       = 80
	DefaultLongPressMs = 1000
	DefaultSwipeMs     = 300
)

// Kind is a frame-bound gesture inside a chain.
type Kind string

const (
	KindTap       Kind = "tap"
	KindLongPress Kind = "long-press"
	KindSwipe     Kind = "swipe"
)

// Spec is one parsed argv step. Coordinates are still caller-space
// (normalized 0-999 or raw pixels) until the CLI maps them.
type Spec struct {
	Kind       Kind
	X          int
	Y          int
	X2         int
	Y2         int
	DurationMs int
}

// Step is a device-pixel gesture ready to inject.
type Step struct {
	Kind       Kind
	X          int
	Y          int
	X2         int
	Y2         int
	DurationMs int
}

// StepError names which chain step failed. Completed steps are not rolled back.
type StepError struct {
	Step      int
	Completed int
	Err       error
}

func (e StepError) Error() string {
	if e.Err == nil {
		return fmt.Sprintf("step %d failed after %d completed", e.Step, e.Completed)
	}
	return fmt.Sprintf("step %d failed after %d completed: %v", e.Step, e.Completed, e.Err)
}

func (e StepError) Unwrap() error { return e.Err }
