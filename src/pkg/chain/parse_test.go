package chain

import (
	"strings"
	"testing"
)

func TestParseTwoTaps(t *testing.T) {
	specs, err := Parse([]string{"tap", "180", "720", "tap", "420", "310"})
	if err != nil {
		t.Fatal(err)
	}
	if len(specs) != 2 {
		t.Fatalf("len = %d", len(specs))
	}
	if specs[0] != (Spec{Kind: KindTap, X: 180, Y: 720}) {
		t.Fatalf("step0 = %+v", specs[0])
	}
	if specs[1] != (Spec{Kind: KindTap, X: 420, Y: 310}) {
		t.Fatalf("step1 = %+v", specs[1])
	}
}

func TestParseMixedGestures(t *testing.T) {
	specs, err := Parse([]string{"long_press", "10", "20", "swipe", "1", "2", "3", "4"})
	if err != nil {
		t.Fatal(err)
	}
	if specs[0].Kind != KindLongPress || specs[0].X != 10 || specs[0].Y != 20 {
		t.Fatalf("long-press = %+v", specs[0])
	}
	if specs[1] != (Spec{Kind: KindSwipe, X: 1, Y: 2, X2: 3, Y2: 4}) {
		t.Fatalf("swipe = %+v", specs[1])
	}
}

func TestParseStepLimits(t *testing.T) {
	if _, err := Parse([]string{"tap", "1", "2"}); err == nil || !strings.Contains(err.Error(), "at least 2") {
		t.Fatalf("one step: %v", err)
	}
	args := make([]string, 0, MaxSteps*3+3)
	for i := 0; i < MaxSteps+1; i++ {
		args = append(args, "tap", "1", "2")
	}
	if _, err := Parse(args); err == nil || !strings.Contains(err.Error(), "at most 8") {
		t.Fatalf("too many: %v", err)
	}
}

func TestParseRejectsUnknownAndShort(t *testing.T) {
	if _, err := Parse([]string{"type", "1", "2", "tap", "3", "4"}); err == nil || !strings.Contains(err.Error(), "unknown") {
		t.Fatalf("type: %v", err)
	}
	if _, err := Parse([]string{"tap", "1"}); err == nil || !strings.Contains(err.Error(), "requires 2") {
		t.Fatalf("short tap: %v", err)
	}
	if _, err := Parse([]string{"swipe", "1", "2", "3"}); err == nil || !strings.Contains(err.Error(), "requires 4") {
		t.Fatalf("short swipe: %v", err)
	}
	if _, err := Parse([]string{"tap", "x", "2", "tap", "3", "4"}); err == nil || !strings.Contains(err.Error(), "invalid coordinate") {
		t.Fatalf("bad int: %v", err)
	}
}
