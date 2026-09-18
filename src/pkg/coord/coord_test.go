package coord

import "testing"

func TestToPixelEdges(t *testing.T) {
	if got := ToPixel(0, 1080); got != 0 {
		t.Fatalf("0 -> %d", got)
	}
	if got := ToPixel(999, 1080); got != 1079 {
		t.Fatalf("999 -> %d, want 1079", got)
	}
	if got := ToPixel(-10, 1080); got != 0 {
		t.Fatalf("clamp low -> %d", got)
	}
	if got := ToPixel(2000, 1080); got != 1079 {
		t.Fatalf("clamp high -> %d", got)
	}
	if got := ToPixel(500, 0); got != 0 {
		t.Fatalf("zero size -> %d", got)
	}
}

func TestRoundTrip(t *testing.T) {
	for _, size := range []int{720, 1080, 2340} {
		for _, n := range []int{0, 1, 250, 500, 749, 999} {
			p := ToPixel(n, size)
			back := FromPixel(p, size)
			if back < n-2 || back > n+2 {
				t.Fatalf("size=%d n=%d pixel=%d back=%d", size, n, p, back)
			}
		}
	}
}

func TestRotationUsesCurrentAxes(t *testing.T) {
	// Landscape 2340x1080: x maps along the long edge.
	p := Denormalize(999, 0, 2340, 1080)
	if p.X != 2339 || p.Y != 0 {
		t.Fatalf("landscape max x = %+v", p)
	}
	p = Denormalize(0, 999, 2340, 1080)
	if p.X != 0 || p.Y != 1079 {
		t.Fatalf("landscape max y = %+v", p)
	}
}

func TestValidateNormalized(t *testing.T) {
	if err := ValidateNormalized("x", 500); err != nil {
		t.Fatal(err)
	}
	if err := ValidateNormalized("x", 1000); err == nil {
		t.Fatal("expected error")
	}
}
