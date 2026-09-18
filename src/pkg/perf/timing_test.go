package perf

import "testing"

func TestPercentile(t *testing.T) {
	samples := []int64{100, 200, 300, 400, 500}
	if got := Percentile(samples, 50); got != 300 {
		t.Errorf("p50 = %d, want 300", got)
	}
	if got := Percentile(samples, 0); got != 100 {
		t.Errorf("p0 = %d, want 100", got)
	}
	if got := Percentile(samples, 100); got != 500 {
		t.Errorf("p100 = %d, want 500", got)
	}
	if got := Percentile(nil, 50); got != 0 {
		t.Errorf("empty percentile = %d, want 0", got)
	}
}

func TestMax(t *testing.T) {
	if got := Max([]int64{1, 9, 3}); got != 9 {
		t.Errorf("max = %d, want 9", got)
	}
	if got := Max(nil); got != 0 {
		t.Errorf("empty max = %d, want 0", got)
	}
}
