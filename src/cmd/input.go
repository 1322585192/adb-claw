package cmd

import (
	"fmt"
	"strconv"
	"time"

	"github.com/llm-net/adb-claw/pkg/adb"
	"github.com/llm-net/adb-claw/pkg/coord"
	"github.com/llm-net/adb-claw/pkg/frameartifact"
	"github.com/llm-net/adb-claw/pkg/input"
	"github.com/llm-net/adb-claw/pkg/observe"
	"github.com/spf13/cobra"
)

var (
	tapNormalized        bool
	tapRaw               bool
	tapFrame             string
	tapWaitChanged       int
	longPressNormalized  bool
	longPressRaw         bool
	longPressFrame       string
	longPressWaitChanged int
	swipeNormalized      bool
	swipeRaw             bool
	swipeFrame           string
	swipeWaitChanged     int
)

var tapCmd = &cobra.Command{
	Use:   "tap <x> <y>",
	Short: "Tap with a frame-bound 0-999 point or explicit raw pixels",
	Long: `Tap a location.
  Device pixels:     adb-claw tap --raw 540 1200
  Normalized 0-999:  adb-claw tap --normalized 500 500 --frame FRAME_TOKEN`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		start := time.Now()
		x, y, meta, err := parsePoint(args[0], args[1], tapNormalized, tapRaw, tapFrame)
		if err != nil {
			failActionPoint("tap", err, start)
			return nil
		}
		if tapWaitChanged > 0 && meta == nil {
			writer.Fail("tap", "INVALID_ARGS", "--wait-changed requires --normalized and --frame", "", start)
			return nil
		}
		writer.Verbose("tapping at (%d, %d)", x, y)
		if err := input.Tap(client, x, y); err != nil {
			writer.Fail("tap", "TAP_FAILED", err.Error(), "", start)
			return nil
		}
		writeActionSuccess("tap", map[string]interface{}{
			"x":           x,
			"y":           y,
			"normalized":  tapNormalized,
			"frame_token": frameToken(meta),
			"method":      "adb_input",
		}, meta, tapWaitChanged, start)
		return nil
	},
}

var (
	longPressDuration int
)

var longPressCmd = &cobra.Command{
	Use:   "long-press <x> <y>",
	Short: "Long press a coordinate (device pixels or --normalized 0-999)",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		start := time.Now()
		x, y, meta, err := parsePoint(args[0], args[1], longPressNormalized, longPressRaw, longPressFrame)
		if err != nil {
			failActionPoint("long-press", err, start)
			return nil
		}
		if longPressWaitChanged > 0 && meta == nil {
			writer.Fail("long-press", "INVALID_ARGS", "--wait-changed requires --normalized and --frame", "", start)
			return nil
		}
		writer.Verbose("long-pressing at (%d, %d) for %dms", x, y, longPressDuration)
		if err := input.LongPress(client, x, y, longPressDuration); err != nil {
			writer.Fail("long-press", "LONG_PRESS_FAILED", err.Error(), "", start)
			return nil
		}
		writeActionSuccess("long-press", map[string]interface{}{
			"x":           x,
			"y":           y,
			"duration_ms": longPressDuration,
			"normalized":  longPressNormalized,
			"frame_token": frameToken(meta),
			"method":      "adb_input",
		}, meta, longPressWaitChanged, start)
		return nil
	},
}

var (
	swipeDuration int
)

var swipeCmd = &cobra.Command{
	Use:   "swipe <x1> <y1> <x2> <y2>",
	Short: "Swipe from one point to another",
	Args:  cobra.ExactArgs(4),
	RunE: func(cmd *cobra.Command, args []string) error {
		start := time.Now()
		x1, y1, meta, err := parsePoint(args[0], args[1], swipeNormalized, swipeRaw, swipeFrame)
		if err != nil {
			failActionPoint("swipe", err, start)
			return nil
		}
		if swipeWaitChanged > 0 && meta == nil {
			writer.Fail("swipe", "INVALID_ARGS", "--wait-changed requires --normalized and --frame", "", start)
			return nil
		}
		x2, y2, err := mapPoint(args[2], args[3], swipeNormalized, meta)
		if err != nil {
			writer.Fail("swipe", "INVALID_ARGS", err.Error(), "", start)
			return nil
		}
		writer.Verbose("swiping (%d,%d) → (%d,%d) in %dms", x1, y1, x2, y2, swipeDuration)
		if err := input.Swipe(client, x1, y1, x2, y2, swipeDuration); err != nil {
			writer.Fail("swipe", "SWIPE_FAILED", err.Error(), "", start)
			return nil
		}
		writeActionSuccess("swipe", map[string]interface{}{
			"x1":          x1,
			"y1":          y1,
			"x2":          x2,
			"y2":          y2,
			"duration_ms": swipeDuration,
			"normalized":  swipeNormalized,
			"frame_token": frameToken(meta),
			"method":      "adb_input",
		}, meta, swipeWaitChanged, start)
		return nil
	},
}

var keyCmd = &cobra.Command{
	Use:   "key <KEY_NAME>",
	Short: "Send a key event (HOME, BACK, ENTER, etc.)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		start := time.Now()
		key := args[0]
		writer.Verbose("sending key event: %s", key)
		if err := input.KeyEvent(client, key); err != nil {
			writer.Fail("key", "KEY_FAILED", err.Error(), "", start)
			return nil
		}
		writer.Success("key", map[string]interface{}{
			"key":    key,
			"method": "adb_input",
		}, start)
		return nil
	},
}

var typeCmd = &cobra.Command{
	Use:   "type <text>",
	Short: "Input text into the focused field",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		start := time.Now()
		text := args[0]
		writer.Verbose("typing text: %q", text)
		method, err := input.TypeText(client, text)
		if err != nil {
			writer.Fail("type", "TYPE_FAILED", err.Error(),
				"Keep the input field focused and retry once; adb-claw handles Unicode without installing an IME", start)
			return nil
		}
		writer.Success("type", map[string]interface{}{
			"text":   text,
			"length": len(text),
			"method": method,
		}, start)
		return nil
	},
}

func parsePoint(xs, ys string, normalized, raw bool, token string) (int, int, *frameartifact.Metadata, error) {
	if normalized == raw {
		return 0, 0, nil, fmt.Errorf("choose exactly one coordinate mode: --normalized with --frame, or --raw")
	}
	if raw {
		x, y, err := mapPoint(xs, ys, false, nil)
		return x, y, nil, err
	}
	if token == "" {
		return 0, 0, nil, staleFrameError{"--normalized requires --frame from the latest observe result"}
	}
	meta, err := loadActionFrame(token)
	if err != nil {
		return 0, 0, nil, err
	}
	x, y, err := mapPoint(xs, ys, true, meta)
	return x, y, meta, err
}

func loadActionFrame(token string) (*frameartifact.Metadata, error) {
	meta, err := frameartifact.Load(token)
	if err != nil {
		return nil, staleFrameError{err.Error()}
	}
	if err := validateFrameActionSpace(client, meta); err != nil {
		return nil, err
	}
	return meta, nil
}

func commanderReady(cmd adb.Commander) bool {
	if cmd == nil {
		return false
	}
	if c, ok := cmd.(*adb.Client); ok {
		return c != nil
	}
	return true
}

func validateFrameActionSpace(cmd adb.Commander, meta *frameartifact.Metadata) error {
	if meta == nil || !commanderReady(cmd) {
		return nil
	}
	w, h, err := input.CurrentScreenSize(cmd)
	if err == nil && w > 0 && h > 0 {
		if input.SameOrientedSize(meta.ActionWidth, meta.ActionHeight, w, h) {
			return nil
		}
		return staleFrameError{"screen orientation or size changed since this frame; observe again"}
	}
	if !meta.RotationKnown {
		return nil
	}
	rotation, rotErr := input.CurrentRotation(cmd)
	if rotErr != nil {
		return staleFrameError{"cannot verify current screen; observe again"}
	}
	if rotation != meta.Rotation {
		return staleFrameError{"screen rotated since this frame; observe again"}
	}
	return nil
}

func mapPoint(xs, ys string, normalized bool, meta *frameartifact.Metadata) (int, int, error) {
	x, err := strconv.Atoi(xs)
	if err != nil {
		return 0, 0, fmt.Errorf("invalid x: %s", xs)
	}
	y, err := strconv.Atoi(ys)
	if err != nil {
		return 0, 0, fmt.Errorf("invalid y: %s", ys)
	}
	if !normalized {
		return x, y, nil
	}
	if err := coord.ValidateNormalized("x", x); err != nil {
		return 0, 0, err
	}
	if err := coord.ValidateNormalized("y", y); err != nil {
		return 0, 0, err
	}
	if meta == nil || meta.ActionWidth <= 0 || meta.ActionHeight <= 0 {
		return 0, 0, fmt.Errorf("valid frame metadata is required")
	}
	p := coord.Denormalize(x, y, meta.ActionWidth, meta.ActionHeight)
	return p.X, p.Y, nil
}

type staleFrameError struct{ message string }

func (e staleFrameError) Error() string { return e.message }

func failActionPoint(command string, err error, start time.Time) {
	code := "INVALID_ARGS"
	suggestion := "Use --normalized X Y --frame TOKEN from the latest observe result; humans may use --raw"
	if _, ok := err.(staleFrameError); ok {
		code = "STALE_FRAME"
		suggestion = "Run adb-claw observe, read its JPEG, then use the returned frame_token"
	}
	writer.Fail(command, code, err.Error(), suggestion, start)
}

func frameToken(meta *frameartifact.Metadata) string {
	if meta == nil {
		return ""
	}
	return meta.Token
}

func writeActionSuccess(command string, data map[string]interface{}, baseline *frameartifact.Metadata, waitMs int, start time.Time) {
	if waitMs > 0 && baseline != nil {
		result, err := observe.WaitForChange(client, baseline, time.Duration(waitMs)*time.Millisecond, 100*time.Millisecond)
		if err != nil {
			data["changed"] = false
			data["next_frame_error"] = err.Error()
		} else {
			data["changed"] = result.Changed
			data["attempts"] = result.Attempts
			data["screenshot"] = result.Screenshot
			data["next"] = "read_path_directly"
		}
	}
	writer.SuccessCompact(command, data, start)
}

func init() {
	tapCmd.Flags().BoolVar(&tapNormalized, "normalized", false, "Treat x y as Gemini 0-999 grid coordinates")
	tapCmd.Flags().BoolVar(&tapRaw, "raw", false, "Treat x y as device pixels (human debugging only)")
	tapCmd.Flags().StringVar(&tapFrame, "frame", "", "Frame token returned by observe (required with --normalized)")
	tapCmd.Flags().IntVar(&tapWaitChanged, "wait-changed", 0, "Wait up to N ms and return the next visual frame")
	longPressCmd.Flags().IntVar(&longPressDuration, "duration", 1000, "Long press duration in ms")
	longPressCmd.Flags().BoolVar(&longPressNormalized, "normalized", false, "Treat x y as Gemini 0-999 grid coordinates")
	longPressCmd.Flags().BoolVar(&longPressRaw, "raw", false, "Treat x y as device pixels (human debugging only)")
	longPressCmd.Flags().StringVar(&longPressFrame, "frame", "", "Frame token returned by observe (required with --normalized)")
	longPressCmd.Flags().IntVar(&longPressWaitChanged, "wait-changed", 0, "Wait up to N ms and return the next visual frame")
	swipeCmd.Flags().IntVar(&swipeDuration, "duration", 300, "Swipe duration in ms")
	swipeCmd.Flags().BoolVar(&swipeNormalized, "normalized", false, "Treat coordinates as Gemini 0-999 grid")
	swipeCmd.Flags().BoolVar(&swipeRaw, "raw", false, "Treat coordinates as device pixels (human debugging only)")
	swipeCmd.Flags().StringVar(&swipeFrame, "frame", "", "Frame token returned by observe (required with --normalized)")
	swipeCmd.Flags().IntVar(&swipeWaitChanged, "wait-changed", 0, "Wait up to N ms and return the next visual frame")

	rootCmd.AddCommand(tapCmd)
	rootCmd.AddCommand(longPressCmd)
	rootCmd.AddCommand(swipeCmd)
	rootCmd.AddCommand(keyCmd)
	rootCmd.AddCommand(typeCmd)
}
