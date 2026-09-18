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

func TestAdaptiveStableStaysNative(t *testing.T) {
	a := NewAdaptive(0, 0)
	for i := 0; i < sampleWindow; i++ {
		if a.Observe(120, 40) {
			t.Fatal("stable session should stay at native width")
		}
	}
	if a.Width() != 0 || a.Locked() {
		t.Fatalf("width=%d locked=%v", a.Width(), a.Locked())
	}
}

func TestAdaptiveDropsNativeTo540AndStays(t *testing.T) {
	a := NewAdaptive(0, QualityHigh)
	if a.Width() != 0 {
		t.Fatalf("start width %d, want native 0", a.Width())
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
}

func TestAdaptiveExplicitLowLocks(t *testing.T) {
	a := NewAdaptive(WidthLow, 0)
	if a.Width() != WidthLow || !a.Locked() {
		t.Fatalf("explicit 540 should lock, got width=%d locked=%v", a.Width(), a.Locked())
	}
}
