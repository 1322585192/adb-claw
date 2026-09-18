package frame

import "testing"

func TestAdaptiveDrops720To540AndStays(t *testing.T) {
	a := NewAdaptive(WidthHigh, QualityHigh)
	if a.Width() != WidthHigh {
		t.Fatalf("start width %d", a.Width())
	}
	dropped := false
	for i := 0; i < sampleWindow; i++ {
		if a.Observe(800, 80) {
			dropped = true
		}
	}
	if !dropped || a.Width() != WidthLow || a.Quality() != QualityLow {
		t.Fatalf("expected drop, width=%d q=%d locked=%v", a.Width(), a.Quality(), a.Locked())
	}
	for i := 0; i < sampleWindow; i++ {
		if a.Observe(50, 10) {
			t.Fatal("must not drop again")
		}
	}
	if a.Width() != WidthLow {
		t.Fatal("must not auto-upgrade after recovery")
	}
}

func TestAdaptiveStableStaysHigh(t *testing.T) {
	a := NewAdaptive(0, 0)
	for i := 0; i < sampleWindow; i++ {
		if a.Observe(120, 40) {
			t.Fatal("stable session should stay at 720")
		}
	}
	if a.Width() != WidthHigh || a.Locked() {
		t.Fatalf("width=%d locked=%v", a.Width(), a.Locked())
	}
}

func TestAdaptiveExplicitLowLocks(t *testing.T) {
	a := NewAdaptive(WidthLow, 0)
	if a.Width() != WidthLow || !a.Locked() {
		t.Fatalf("explicit 540 should lock, got width=%d locked=%v", a.Width(), a.Locked())
	}
}
