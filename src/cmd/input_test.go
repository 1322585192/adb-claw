package cmd

import (
	"fmt"
	"testing"
	"time"

	"github.com/llm-net/adb-claw/pkg/adb"
	"github.com/llm-net/adb-claw/pkg/frameartifact"
)

func saveTestFrame(t *testing.T, width, height int, hash string) *frameartifact.Metadata {
	t.Helper()
	token := frameartifact.NewToken()
	meta := &frameartifact.Metadata{
		Token:        token,
		Hash:         hash,
		CapturedAt:   time.Now().UTC(),
		Path:         frameartifact.Path(token, "jpeg"),
		Format:       "jpeg",
		DeviceWidth:  width,
		DeviceHeight: height,
		ActionWidth:  width,
		ActionHeight: height,
		ImageWidth:   width,
		ImageHeight:  height,
		Complete:     true,
	}
	if err := frameartifact.Save(*meta); err != nil {
		t.Fatal(err)
	}
	return meta
}

func TestParsePointRequiresFrameForNormalized(t *testing.T) {
	t.Setenv("TMPDIR", t.TempDir())
	_, _, _, err := parsePoint("500", "500", true, false, "")
	if _, ok := err.(staleFrameError); !ok {
		t.Fatalf("error = %T %v, want staleFrameError", err, err)
	}
}

func TestParsePointUsesFrameLandscapeSpace(t *testing.T) {
	t.Setenv("TMPDIR", t.TempDir())
	meta := saveTestFrame(t, 2780, 1264, "landscape")
	x, y, gotMeta, err := parsePoint("500", "750", true, false, meta.Token)
	if err != nil {
		t.Fatal(err)
	}
	if x != 1390 || y != 948 {
		t.Fatalf("point = (%d,%d), want (1390,948)", x, y)
	}
	if gotMeta.Token != meta.Token {
		t.Fatalf("frame token = %q", gotMeta.Token)
	}
}

func TestParsePointRawMustBeExplicit(t *testing.T) {
	if _, _, _, err := parsePoint("10", "20", false, false, ""); err == nil {
		t.Fatal("bare pixel coordinates must be rejected")
	}
	x, y, meta, err := parsePoint("10", "20", false, true, "")
	if err != nil || x != 10 || y != 20 || meta != nil {
		t.Fatalf("raw point = (%d,%d,%v,%v)", x, y, meta, err)
	}
}

type rotationCommander struct{ output string }

func (c *rotationCommander) Shell(args ...string) (*adb.Result, error) {
	return &adb.Result{Stdout: c.output}, nil
}
func (c *rotationCommander) ExecOut(args ...string) ([]byte, error) {
	return nil, fmt.Errorf("not implemented")
}
func (c *rotationCommander) RawCommand(args ...string) (*adb.Result, error) {
	return nil, fmt.Errorf("not implemented")
}

func TestValidateFrameRotationRejectsOldToken(t *testing.T) {
	meta := &frameartifact.Metadata{Rotation: 1, RotationKnown: true}
	if err := validateFrameRotation(&rotationCommander{output: "mCurrentRotation=1"}, meta); err != nil {
		t.Fatalf("matching rotation: %v", err)
	}
	err := validateFrameRotation(&rotationCommander{output: "mCurrentRotation=0"}, meta)
	if _, ok := err.(staleFrameError); !ok {
		t.Fatalf("rotation change error = %T %v, want staleFrameError", err, err)
	}
}
