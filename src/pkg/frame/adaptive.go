package frame

import (
	"github.com/llm-net/adb-claw/pkg/perf"
)

const (
	WidthHigh = 720
	WidthLow  = 540

	QualityHigh = 60
	QualityLow  = 50

	defaultIntervalMs = 250
	sampleWindow      = 8
	ageP95LimitMs     = 700
	transferP95LimitMs = 500
)

// Adaptive starts at the native display (width 0) and may cap width at
// WidthLow when frames are stale. Scaling is always uniform, so the
// device aspect ratio is kept. It never auto-upgrades.
type Adaptive struct {
	width   int
	quality int
	locked  bool
	ages    []int64
	xfers   []int64
}

// NewAdaptive starts at native size (width 0) / quality 60 unless an
// explicit positive width is given. Width <= WidthLow locks at 540.
func NewAdaptive(width, quality int) *Adaptive {
	a := &Adaptive{width: 0, quality: QualityHigh}
	if width > 0 && width <= WidthLow {
		a.width = WidthLow
		a.quality = QualityLow
		a.locked = true
	} else if width > 0 {
		a.width = width
	}
	if quality > 0 && (a.width == 0 || a.width >= WidthHigh) {
		a.quality = quality
	}
	return a
}

// Width is the current target encode width.
func (a *Adaptive) Width() int { return a.width }

// Quality is the current JPEG quality.
func (a *Adaptive) Quality() int { return a.quality }

// Locked reports whether the session already dropped and will stay low.
func (a *Adaptive) Locked() bool { return a.locked }

// Observe records one frame's age and transfer latency and may drop resolution.
func (a *Adaptive) Observe(ageMs, transferMs int64) (dropped bool) {
	if a == nil || a.locked {
		return false
	}
	a.ages = append(a.ages, ageMs)
	a.xfers = append(a.xfers, transferMs)
	if len(a.ages) > sampleWindow {
		a.ages = a.ages[len(a.ages)-sampleWindow:]
		a.xfers = a.xfers[len(a.xfers)-sampleWindow:]
	}
	if len(a.ages) < sampleWindow {
		return false
	}
	if perf.Percentile(a.ages, 95) > ageP95LimitMs || perf.Percentile(a.xfers, 95) > transferP95LimitMs {
		a.width = WidthLow
		a.quality = QualityLow
		a.locked = true
		return true
	}
	return false
}
