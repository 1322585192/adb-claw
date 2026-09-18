package cmd

import (
	"fmt"
	"strconv"
	"time"

	"github.com/llm-net/adb-claw/pkg/coord"
	"github.com/llm-net/adb-claw/pkg/input"
	"github.com/spf13/cobra"
)

var (
	tapNormalized       bool
	longPressNormalized bool
	swipeNormalized     bool
)

var tapCmd = &cobra.Command{
	Use:   "tap <x> <y>",
	Short: "Tap a coordinate (device pixels or --normalized 0-999)",
	Long: `Tap a location.
  Device pixels:     adb-claw tap 540 1200
  Normalized 0-999:  adb-claw tap --normalized 500 500`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		start := time.Now()
		x, y, err := parsePoint(args[0], args[1], tapNormalized)
		if err != nil {
			writer.Fail("tap", "INVALID_ARGS", err.Error(), "Example: adb-claw tap --normalized 500 500", start)
			return nil
		}
		writer.Verbose("tapping at (%d, %d)", x, y)
		if err := input.Tap(client, x, y); err != nil {
			writer.Fail("tap", "TAP_FAILED", err.Error(), "", start)
			return nil
		}
		writer.Success("tap", map[string]interface{}{
			"x":          x,
			"y":          y,
			"normalized": tapNormalized,
			"method":     "adb_input",
		}, start)
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
		x, y, err := parsePoint(args[0], args[1], longPressNormalized)
		if err != nil {
			writer.Fail("long-press", "INVALID_ARGS", err.Error(), "", start)
			return nil
		}
		writer.Verbose("long-pressing at (%d, %d) for %dms", x, y, longPressDuration)
		if err := input.LongPress(client, x, y, longPressDuration); err != nil {
			writer.Fail("long-press", "LONG_PRESS_FAILED", err.Error(), "", start)
			return nil
		}
		writer.Success("long-press", map[string]interface{}{
			"x":           x,
			"y":           y,
			"duration_ms": longPressDuration,
			"normalized":  longPressNormalized,
			"method":      "adb_input",
		}, start)
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
		x1, y1, err := parsePoint(args[0], args[1], swipeNormalized)
		if err != nil {
			writer.Fail("swipe", "INVALID_ARGS", err.Error(), "", start)
			return nil
		}
		x2, y2, err := parsePoint(args[2], args[3], swipeNormalized)
		if err != nil {
			writer.Fail("swipe", "INVALID_ARGS", err.Error(), "", start)
			return nil
		}
		writer.Verbose("swiping (%d,%d) → (%d,%d) in %dms", x1, y1, x2, y2, swipeDuration)
		if err := input.Swipe(client, x1, y1, x2, y2, swipeDuration); err != nil {
			writer.Fail("swipe", "SWIPE_FAILED", err.Error(), "", start)
			return nil
		}
		writer.Success("swipe", map[string]interface{}{
			"x1":          x1,
			"y1":          y1,
			"x2":          x2,
			"y2":          y2,
			"duration_ms": swipeDuration,
			"normalized":  swipeNormalized,
			"method":      "adb_input",
		}, start)
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
		if err := input.TypeText(client, text); err != nil {
			writer.Fail("type", "TYPE_FAILED", err.Error(),
				"Ensure an input field is focused first", start)
			return nil
		}
		writer.Success("type", map[string]interface{}{
			"text":   text,
			"length": len(text),
			"method": "adb_input",
		}, start)
		return nil
	},
}

func parsePoint(xs, ys string, normalized bool) (int, int, error) {
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
	w, h, err := input.CurrentScreenSize(client)
	if err != nil {
		return 0, 0, err
	}
	p := coord.Denormalize(x, y, w, h)
	return p.X, p.Y, nil
}

func init() {
	tapCmd.Flags().BoolVar(&tapNormalized, "normalized", false, "Treat x y as Gemini 0-999 grid coordinates")
	longPressCmd.Flags().IntVar(&longPressDuration, "duration", 1000, "Long press duration in ms")
	longPressCmd.Flags().BoolVar(&longPressNormalized, "normalized", false, "Treat x y as Gemini 0-999 grid coordinates")
	swipeCmd.Flags().IntVar(&swipeDuration, "duration", 300, "Swipe duration in ms")
	swipeCmd.Flags().BoolVar(&swipeNormalized, "normalized", false, "Treat coordinates as Gemini 0-999 grid")

	rootCmd.AddCommand(tapCmd)
	rootCmd.AddCommand(longPressCmd)
	rootCmd.AddCommand(swipeCmd)
	rootCmd.AddCommand(keyCmd)
	rootCmd.AddCommand(typeCmd)
}
