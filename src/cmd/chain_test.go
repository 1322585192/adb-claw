package cmd

import (
	"testing"

	"github.com/llm-net/adb-claw/pkg/chain"
	"github.com/llm-net/adb-claw/pkg/coord"
)

func TestMapChainSpecsTwoTaps(t *testing.T) {
	t.Setenv("TMPDIR", t.TempDir())
	meta := saveTestFrame(t, 2780, 1264, "board")
	specs := []chain.Spec{
		{Kind: chain.KindTap, X: 180, Y: 720},
		{Kind: chain.KindTap, X: 420, Y: 310},
	}
	steps, reports, got, err := mapChainSpecs(specs, true, false, meta.Token)
	if err != nil {
		t.Fatal(err)
	}
	if got.Token != meta.Token {
		t.Fatalf("token = %q", got.Token)
	}
	want0 := coord.Denormalize(180, 720, 2780, 1264)
	want1 := coord.Denormalize(420, 310, 2780, 1264)
	if steps[0] != (chain.Step{Kind: chain.KindTap, X: want0.X, Y: want0.Y}) {
		t.Fatalf("step0 = %+v, want %+v", steps[0], want0)
	}
	if steps[1] != (chain.Step{Kind: chain.KindTap, X: want1.X, Y: want1.Y}) {
		t.Fatalf("step1 = %+v, want %+v", steps[1], want1)
	}
	if reports[0]["x"] != 180 || reports[1]["device_x"] != want1.X {
		t.Fatalf("reports = %#v", reports)
	}
}

func TestMapChainSpecsSwipeUsesSameFrame(t *testing.T) {
	t.Setenv("TMPDIR", t.TempDir())
	meta := saveTestFrame(t, 1080, 2400, "swipe")
	specs := []chain.Spec{
		{Kind: chain.KindTap, X: 100, Y: 200},
		{Kind: chain.KindSwipe, X: 100, Y: 800, X2: 100, Y2: 200},
	}
	steps, _, _, err := mapChainSpecs(specs, true, false, meta.Token)
	if err != nil {
		t.Fatal(err)
	}
	end := coord.Denormalize(100, 200, 1080, 2400)
	if steps[1].X2 != end.X || steps[1].Y2 != end.Y {
		t.Fatalf("swipe end = (%d,%d), want (%d,%d)", steps[1].X2, steps[1].Y2, end.X, end.Y)
	}
}

func TestMapChainSpecsRequiresFrame(t *testing.T) {
	specs := []chain.Spec{
		{Kind: chain.KindTap, X: 1, Y: 2},
		{Kind: chain.KindTap, X: 3, Y: 4},
	}
	_, _, _, err := mapChainSpecs(specs, true, false, "")
	if _, ok := err.(staleFrameError); !ok {
		t.Fatalf("error = %T %v, want staleFrameError", err, err)
	}
}

func TestMapChainSpecsRejectsBareCoords(t *testing.T) {
	specs := []chain.Spec{
		{Kind: chain.KindTap, X: 1, Y: 2},
		{Kind: chain.KindTap, X: 3, Y: 4},
	}
	if _, _, _, err := mapChainSpecs(specs, false, false, ""); err == nil {
		t.Fatal("bare coordinates must be rejected")
	}
}
