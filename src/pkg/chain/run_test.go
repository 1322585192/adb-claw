package chain

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/llm-net/adb-claw/pkg/adb"
)

type recCmd struct {
	shells [][]string
	execs  int
	failAt int
}

func (c *recCmd) Shell(args ...string) (*adb.Result, error) {
	copied := append([]string{}, args...)
	c.shells = append(c.shells, copied)
	if c.failAt > 0 && len(c.shells) == c.failAt {
		return &adb.Result{Stderr: "denied", ExitCode: 1}, nil
	}
	return &adb.Result{}, nil
}

func (c *recCmd) ExecOut(args ...string) ([]byte, error) {
	c.execs++
	return nil, fmt.Errorf("observe/screencap must not run mid-chain: %v", args)
}

func (c *recCmd) RawCommand(args ...string) (*adb.Result, error) {
	return nil, fmt.Errorf("unexpected RawCommand: %v", args)
}

func TestRunTwoTapsNoObserve(t *testing.T) {
	cmd := &recCmd{}
	err := Run(cmd, []Step{
		{Kind: KindTap, X: 100, Y: 200},
		{Kind: KindTap, X: 300, Y: 400},
	}, 0)
	if err != nil {
		t.Fatal(err)
	}
	if cmd.execs != 0 {
		t.Fatalf("ExecOut called %d times", cmd.execs)
	}
	if len(cmd.shells) != 2 {
		t.Fatalf("shells = %v", cmd.shells)
	}
	assertShell(t, cmd.shells[0], "input", "tap", "100", "200")
	assertShell(t, cmd.shells[1], "input", "tap", "300", "400")
}

func TestRunSwipeAndLongPress(t *testing.T) {
	cmd := &recCmd{}
	err := Run(cmd, []Step{
		{Kind: KindLongPress, X: 10, Y: 20, DurationMs: 500},
		{Kind: KindSwipe, X: 1, Y: 2, X2: 3, Y2: 4, DurationMs: 200},
	}, 0)
	if err != nil {
		t.Fatal(err)
	}
	assertShell(t, cmd.shells[0], "input", "swipe", "10", "20", "10", "20", "500")
	assertShell(t, cmd.shells[1], "input", "swipe", "1", "2", "3", "4", "200")
}

func TestRunReportsFailedStep(t *testing.T) {
	cmd := &recCmd{failAt: 2}
	err := Run(cmd, []Step{
		{Kind: KindTap, X: 1, Y: 1},
		{Kind: KindTap, X: 2, Y: 2},
		{Kind: KindTap, X: 3, Y: 3},
	}, 0)
	var stepErr StepError
	if !errors.As(err, &stepErr) {
		t.Fatalf("error = %T %v", err, err)
	}
	if stepErr.Step != 2 || stepErr.Completed != 1 {
		t.Fatalf("step error = %+v", stepErr)
	}
	if len(cmd.shells) != 2 {
		t.Fatalf("injected %d steps after failure", len(cmd.shells))
	}
}

func TestRunRejectsEmpty(t *testing.T) {
	if err := Run(&recCmd{}, nil, 0); err == nil || !strings.Contains(err.Error(), "no steps") {
		t.Fatalf("empty: %v", err)
	}
}

func assertShell(t *testing.T, got []string, want ...string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("shell %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("shell %v, want %v", got, want)
		}
	}
}
