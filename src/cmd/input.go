package cmd

import (
	"fmt"
	"strconv"
	"time"

	"github.com/llm-net/adb-claw/pkg/input"
	"github.com/spf13/cobra"
)

var (
	tapIndex        int
	tapID           string
	tapText         string
	refreshElements bool
)

var tapCmd = &cobra.Command{
	Use:   "tap [x y]",
	Short: "Tap on a coordinate or UI element",
	Long: `Tap on a specific location. Can target by:
  - Coordinates: adb-claw tap 540 1200
  - Element index: adb-claw tap --index 3
  - Resource ID: adb-claw tap --id "btn_login"
  - Text content: adb-claw tap --text "Login"`,
	RunE: func(cmd *cobra.Command, args []string) error {
		start := time.Now()

		// Determine target coordinates
		var x, y int
		var targetInfo map[string]interface{}

		switch {
		case cmd.Flags().Changed("index"):
			el, err := resolveElementByIndex(tapIndex)
			if err != nil {
				writer.Fail("tap", resolveErrorCode(err), err.Error(),
					"Run observe first, pass coordinates, or use --refresh", start)
				return nil
			}
			x, y = el.Center.X, el.Center.Y
			targetInfo = elementInfo(el)

		case tapID != "":
			el, err := resolveElementByID(tapID)
			if err != nil {
				writer.Fail("tap", resolveErrorCode(err), err.Error(),
					"Run observe first, pass coordinates, or use --refresh", start)
				return nil
			}
			x, y = el.Center.X, el.Center.Y
			targetInfo = elementInfo(el)

		case tapText != "":
			el, err := resolveElementByText(tapText)
			if err != nil {
				writer.Fail("tap", resolveErrorCode(err), err.Error(),
					"Run observe first, pass coordinates, or use --refresh", start)
				return nil
			}
			x, y = el.Center.X, el.Center.Y
			targetInfo = elementInfo(el)

		default:
			if len(args) < 2 {
				writer.Fail("tap", "MISSING_ARGS",
					"Specify coordinates (x y) or use --index/--id/--text",
					"Example: adb-claw tap 540 1200", start)
				return nil
			}
			var err error
			x, err = strconv.Atoi(args[0])
			if err != nil {
				writer.Fail("tap", "INVALID_ARGS", "Invalid x coordinate: "+args[0], "", start)
				return nil
			}
			y, err = strconv.Atoi(args[1])
			if err != nil {
				writer.Fail("tap", "INVALID_ARGS", "Invalid y coordinate: "+args[1], "", start)
				return nil
			}
		}

		writer.Verbose("tapping at (%d, %d)", x, y)
		if err := input.Tap(client, x, y); err != nil {
			writer.Fail("tap", "TAP_FAILED", err.Error(), "", start)
			return nil
		}

		data := map[string]interface{}{
			"x":      x,
			"y":      y,
			"method": "adb_input",
		}
		if targetInfo != nil {
			data["element"] = targetInfo
		}
		writer.Success("tap", data, start)
		return nil
	},
}

var (
	longPressDuration int
	longPressIndex    int
	longPressID       string
	longPressText     string
)

var longPressCmd = &cobra.Command{
	Use:   "long-press [x y]",
	Short: "Long press at a coordinate or UI element",
	Long: `Long press on a specific location. Can target by:
  - Coordinates: adb-claw long-press 540 1200
  - Element index: adb-claw long-press --index 3
  - Resource ID: adb-claw long-press --id "btn_login"
  - Text content: adb-claw long-press --text "Login"`,
	RunE: func(cmd *cobra.Command, args []string) error {
		start := time.Now()

		var x, y int
		var targetInfo map[string]interface{}

		switch {
		case cmd.Flags().Changed("index"):
			el, err := resolveElementByIndex(longPressIndex)
			if err != nil {
				writer.Fail("long-press", resolveErrorCode(err), err.Error(),
					"Run observe first, pass coordinates, or use --refresh", start)
				return nil
			}
			x, y = el.Center.X, el.Center.Y
			targetInfo = elementInfo(el)

		case longPressID != "":
			el, err := resolveElementByID(longPressID)
			if err != nil {
				writer.Fail("long-press", resolveErrorCode(err), err.Error(),
					"Run observe first, pass coordinates, or use --refresh", start)
				return nil
			}
			x, y = el.Center.X, el.Center.Y
			targetInfo = elementInfo(el)

		case longPressText != "":
			el, err := resolveElementByText(longPressText)
			if err != nil {
				writer.Fail("long-press", resolveErrorCode(err), err.Error(),
					"Run observe first, pass coordinates, or use --refresh", start)
				return nil
			}
			x, y = el.Center.X, el.Center.Y
			targetInfo = elementInfo(el)

		default:
			if len(args) < 2 {
				writer.Fail("long-press", "MISSING_ARGS",
					"Specify coordinates (x y) or use --index/--id/--text",
					"Example: adb-claw long-press 540 1200", start)
				return nil
			}
			var err error
			x, err = strconv.Atoi(args[0])
			if err != nil {
				writer.Fail("long-press", "INVALID_ARGS", "Invalid x: "+args[0], "", start)
				return nil
			}
			y, err = strconv.Atoi(args[1])
			if err != nil {
				writer.Fail("long-press", "INVALID_ARGS", "Invalid y: "+args[1], "", start)
				return nil
			}
		}

		writer.Verbose("long-pressing at (%d, %d) for %dms", x, y, longPressDuration)
		if err := input.LongPress(client, x, y, longPressDuration); err != nil {
			writer.Fail("long-press", "LONG_PRESS_FAILED", err.Error(), "", start)
			return nil
		}

		data := map[string]interface{}{
			"x":           x,
			"y":           y,
			"duration_ms": longPressDuration,
			"method":      "adb_input",
		}
		if targetInfo != nil {
			data["element"] = targetInfo
		}
		writer.Success("long-press", data, start)
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
		coords := make([]int, 4)
		names := []string{"x1", "y1", "x2", "y2"}
		for i := 0; i < 4; i++ {
			v, err := strconv.Atoi(args[i])
			if err != nil {
				writer.Fail("swipe", "INVALID_ARGS",
					fmt.Sprintf("Invalid %s: %s", names[i], args[i]), "", start)
				return nil
			}
			coords[i] = v
		}

		writer.Verbose("swiping (%d,%d) → (%d,%d) in %dms",
			coords[0], coords[1], coords[2], coords[3], swipeDuration)
		if err := input.Swipe(client, coords[0], coords[1], coords[2], coords[3], swipeDuration); err != nil {
			writer.Fail("swipe", "SWIPE_FAILED", err.Error(), "", start)
			return nil
		}

		writer.Success("swipe", map[string]interface{}{
			"x1":          coords[0],
			"y1":          coords[1],
			"x2":          coords[2],
			"y2":          coords[3],
			"duration_ms": swipeDuration,
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

func init() {
	tapCmd.Flags().IntVar(&tapIndex, "index", -1, "Tap element by UI tree index from the last observe snapshot")
	tapCmd.Flags().StringVar(&tapID, "id", "", "Tap element by resource-id from the last observe snapshot")
	tapCmd.Flags().StringVar(&tapText, "text", "", "Tap element by text content from the last observe snapshot")
	tapCmd.Flags().BoolVar(&refreshElements, "refresh", false, "Re-dump the UI tree instead of using the last observe snapshot")

	longPressCmd.Flags().IntVar(&longPressDuration, "duration", 1000, "Long press duration in ms")
	longPressCmd.Flags().IntVar(&longPressIndex, "index", -1, "Long press element by UI tree index from the last observe snapshot")
	longPressCmd.Flags().StringVar(&longPressID, "id", "", "Long press element by resource-id from the last observe snapshot")
	longPressCmd.Flags().StringVar(&longPressText, "text", "", "Long press element by text content from the last observe snapshot")
	longPressCmd.Flags().BoolVar(&refreshElements, "refresh", false, "Re-dump the UI tree instead of using the last observe snapshot")

	swipeCmd.Flags().IntVar(&swipeDuration, "duration", 300, "Swipe duration in ms")

	rootCmd.AddCommand(tapCmd)
	rootCmd.AddCommand(longPressCmd)
	rootCmd.AddCommand(swipeCmd)
	rootCmd.AddCommand(keyCmd)
	rootCmd.AddCommand(typeCmd)
}
