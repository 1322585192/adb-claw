package cmd

import (
	"fmt"
	"strconv"
	"time"

	"github.com/llm-net/adb-claw/pkg/chain"
	"github.com/llm-net/adb-claw/pkg/frameartifact"
	"github.com/spf13/cobra"
)

var (
	chainNormalized  bool
	chainRaw         bool
	chainFrame       string
	chainWaitChanged int
	chainGapMs       int
)

var chainCmd = &cobra.Command{
	Use:   "chain tap X Y tap X Y ...",
	Short: "Run frame-bound actions on one JPEG without re-observing",
	Long: `Run 2-8 gestures mapped from a single observe frame.

  adb-claw chain --normalized --frame TOKEN --wait-changed 1500 tap 180 720 tap 420 310

All coordinates use that frame's action space. --wait-changed runs once after
the last step. --gap-ms is the inject pause between gestures, not an agent sleep.`,
	Args: cobra.MinimumNArgs(6),
	RunE: runChain,
}

func runChain(cmd *cobra.Command, args []string) error {
	start := time.Now()
	specs, err := chain.Parse(args)
	if err != nil {
		writer.Fail("chain", "INVALID_ARGS", err.Error(),
			"Example: adb-claw chain --normalized --frame TOKEN tap 180 720 tap 420 310", start)
		return nil
	}
	if chainWaitChanged > 0 && !chainNormalized {
		writer.Fail("chain", "INVALID_ARGS", "--wait-changed requires --normalized and --frame", "", start)
		return nil
	}
	steps, reports, meta, err := mapChainSpecs(specs, chainNormalized, chainRaw, chainFrame)
	if err != nil {
		failChain(err, start)
		return nil
	}
	if err := chain.Run(client, steps, time.Duration(chainGapMs)*time.Millisecond); err != nil {
		failChain(err, start)
		return nil
	}
	writeActionSuccess("chain", chainSuccessData(reports, meta), meta, chainWaitChanged, start)
	return nil
}

func mapChainSpecs(specs []chain.Spec, normalized, raw bool, token string) ([]chain.Step, []map[string]interface{}, *frameartifact.Metadata, error) {
	if normalized == raw {
		return nil, nil, nil, fmt.Errorf("choose exactly one coordinate mode: --normalized with --frame, or --raw")
	}
	var meta *frameartifact.Metadata
	if normalized {
		loaded, err := loadActionFrame(token)
		if err != nil {
			return nil, nil, nil, err
		}
		meta = loaded
	}
	steps := make([]chain.Step, 0, len(specs))
	reports := make([]map[string]interface{}, 0, len(specs))
	for _, spec := range specs {
		step, report, err := mapChainSpec(spec, normalized, meta)
		if err != nil {
			return nil, nil, nil, err
		}
		steps = append(steps, step)
		reports = append(reports, report)
	}
	return steps, reports, meta, nil
}

func mapChainSpec(spec chain.Spec, normalized bool, meta *frameartifact.Metadata) (chain.Step, map[string]interface{}, error) {
	x, y, err := mapPoint(strconv.Itoa(spec.X), strconv.Itoa(spec.Y), normalized, meta)
	if err != nil {
		return chain.Step{}, nil, err
	}
	step := chain.Step{Kind: spec.Kind, X: x, Y: y, DurationMs: spec.DurationMs}
	if spec.Kind == chain.KindSwipe {
		x2, y2, err := mapPoint(strconv.Itoa(spec.X2), strconv.Itoa(spec.Y2), normalized, meta)
		if err != nil {
			return chain.Step{}, nil, err
		}
		step.X2, step.Y2 = x2, y2
	}
	return step, chainReport(spec, step), nil
}

func chainReport(spec chain.Spec, step chain.Step) map[string]interface{} {
	report := map[string]interface{}{
		"kind":     spec.Kind,
		"x":        spec.X,
		"y":        spec.Y,
		"device_x": step.X,
		"device_y": step.Y,
	}
	if spec.Kind == chain.KindSwipe {
		report["x2"] = spec.X2
		report["y2"] = spec.Y2
		report["device_x2"] = step.X2
		report["device_y2"] = step.Y2
	}
	return report
}

func chainSuccessData(reports []map[string]interface{}, meta *frameartifact.Metadata) map[string]interface{} {
	return map[string]interface{}{
		"steps":       reports,
		"normalized":  chainNormalized,
		"frame_token": frameToken(meta),
		"gap_ms":      chainGapMs,
		"method":      "adb_input",
	}
}

func failChain(err error, start time.Time) {
	code := "INVALID_ARGS"
	suggestion := "Use --normalized --frame TOKEN tap X Y tap X Y from the latest observe JPEG"
	if _, ok := err.(staleFrameError); ok {
		code = "STALE_FRAME"
		suggestion = "Run adb-claw observe, read its JPEG, then chain with that frame_token"
	}
	if stepErr, ok := err.(chain.StepError); ok {
		code = "CHAIN_FAILED"
		suggestion = fmt.Sprintf("step %d failed after %d completed; observe and decide from the latest JPEG", stepErr.Step, stepErr.Completed)
	}
	writer.Fail("chain", code, err.Error(), suggestion, start)
}

func init() {
	chainCmd.Flags().BoolVar(&chainNormalized, "normalized", false, "Treat coordinates as Gemini 0-999 grid")
	chainCmd.Flags().BoolVar(&chainRaw, "raw", false, "Treat coordinates as device pixels (human debugging only)")
	chainCmd.Flags().StringVar(&chainFrame, "frame", "", "Frame token returned by observe (required with --normalized)")
	chainCmd.Flags().IntVar(&chainWaitChanged, "wait-changed", 0, "After the last step, wait up to N ms for a new visual frame")
	chainCmd.Flags().IntVar(&chainGapMs, "gap-ms", chain.DefaultGapMs, "Inject pause between gestures in ms")
	rootCmd.AddCommand(chainCmd)
}
