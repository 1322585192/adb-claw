package cmd

import (
	"time"

	"github.com/llm-net/adb-claw/pkg/input"
	"github.com/spf13/cobra"
)

var (
	scrollPages    int
	scrollDistance int
	scrollDuration int
)

var scrollCmd = &cobra.Command{
	Use:   "scroll <direction>",
	Short: "Scroll the screen",
	Long: `Scroll in a direction: up, down, left, right.
Examples:
  adb-claw scroll down
  adb-claw scroll up --pages 3
  adb-claw scroll left --distance 500`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		start := time.Now()
		direction := args[0]
		pages := scrollPages
		if pages < 1 {
			pages = 1
		}

		screenW, screenH, err := input.CurrentScreenSize(client)
		if err != nil {
			writer.Fail("scroll", "SCREEN_SIZE_FAILED", err.Error(),
				"Ensure device is connected", start)
			return nil
		}

		dist := scrollDistance
		if dist == 0 {
			switch direction {
			case "up", "down":
				dist = screenH * 60 / 100
			case "left", "right":
				dist = screenW * 60 / 100
			}
		}

		x1, y1, x2, y2, err := input.ScrollDirection(screenW, screenH, dist, direction)
		if err != nil {
			writer.Fail("scroll", "INVALID_DIRECTION", err.Error(), "", start)
			return nil
		}

		var totalScrolled int
		for i := 0; i < pages; i++ {
			writer.Verbose("scroll %s page %d/%d: (%d,%d) → (%d,%d)", direction, i+1, pages, x1, y1, x2, y2)
			if err := input.Swipe(client, x1, y1, x2, y2, scrollDuration); err != nil {
				writer.Fail("scroll", "SWIPE_FAILED", err.Error(), "", start)
				return nil
			}
			dy := y1 - y2
			dx := x1 - x2
			if dy < 0 {
				dy = -dy
			}
			if dx < 0 {
				dx = -dx
			}
			if dy > dx {
				totalScrolled += dy
			} else {
				totalScrolled += dx
			}
			if i < pages-1 {
				time.Sleep(300 * time.Millisecond)
			}
		}

		writer.Success("scroll", map[string]interface{}{
			"direction":       direction,
			"pages":           pages,
			"distance_pixels": totalScrolled,
			"method":          "adb_swipe",
		}, start)
		return nil
	},
}

func init() {
	scrollCmd.Flags().IntVar(&scrollPages, "pages", 1, "Number of pages to scroll")
	scrollCmd.Flags().IntVar(&scrollDistance, "distance", 0, "Scroll distance in pixels (0 = auto)")
	scrollCmd.Flags().IntVar(&scrollDuration, "duration", 300, "Swipe duration in ms")
	rootCmd.AddCommand(scrollCmd)
}
