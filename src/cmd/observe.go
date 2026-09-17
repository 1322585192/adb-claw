package cmd

import (
	"strings"
	"time"

	"github.com/llm-net/adb-claw/pkg/observe"
	"github.com/spf13/cobra"
)

var (
	observeMaxWidth int
	observeFormat   string
	observeQuality  int
	observeFile     string
	observeInline   bool
)

var observeCmd = &cobra.Command{
	Use:   "observe",
	Short: "Capture screenshot + UI tree + device state",
	Long: `Captures a screenshot and UI element tree in parallel, returning both in a single response.

The screenshot is JPEG-compressed and written to a file by default (no base64 in JSON)
to keep agent context small. UI tree bounds/center stay in device pixels — never tap
using preview-image pixels even if --width scaled the screenshot.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		start := time.Now()
		writer.Verbose("starting observe (screenshot + ui tree in parallel)")

		format := strings.ToLower(observeFormat)
		if format != "jpeg" && format != "png" {
			writer.Fail("observe", "INVALID_ARGS",
				"Unsupported screenshot format '"+observeFormat+"'",
				"Use --format jpeg or --format png", start)
			return nil
		}

		result := observe.Observe(client, observe.ObserveOptions{
			MaxWidth: observeMaxWidth,
			Format:   format,
			Quality:  observeQuality,
			Path:     observeFile,
			Inline:   observeInline,
		})

		// Check if we got at least something
		if result.Screenshot == nil && result.UI == nil {
			writer.Fail("observe", "OBSERVE_FAILED",
				"Both screenshot and UI tree failed",
				"Check device connection: adb-claw device list", start)
			return nil
		}

		writer.SuccessCompact("observe", result, start)
		return nil
	},
}

var (
	screenshotOutput   string
	screenshotMaxWidth int
	screenshotFormat   string
	screenshotQuality  int
)

var screenshotCmd = &cobra.Command{
	Use:   "screenshot",
	Short: "Capture a screenshot",
	RunE: func(cmd *cobra.Command, args []string) error {
		start := time.Now()
		writer.Verbose("capturing screenshot")

		format := strings.ToLower(screenshotFormat)
		if format != "jpeg" && format != "png" {
			writer.Fail("screenshot", "INVALID_ARGS",
				"Unsupported screenshot format '"+screenshotFormat+"'",
				"Use --format jpeg or --format png", start)
			return nil
		}

		inline := screenshotOutput == ""
		result, err := observe.CaptureScreenshot(client, observe.CaptureOptions{
			MaxWidth: screenshotMaxWidth,
			Format:   format,
			Quality:  screenshotQuality,
			Path:     screenshotOutput,
			Inline:   inline,
		})
		if err != nil {
			writer.Fail("screenshot", "SCREENSHOT_FAILED", err.Error(),
				"Ensure the device screen is on and unlocked", start)
			return nil
		}

		writer.Success("screenshot", result, start)
		return nil
	},
}

func init() {
	screenshotCmd.Flags().StringVarP(&screenshotOutput, "file", "f", "", "Save screenshot to file instead of base64 output")
	screenshotCmd.Flags().IntVar(&screenshotMaxWidth, "width", 0, "Max image width in pixels (0 = original size; tap coords stay device pixels)")
	screenshotCmd.Flags().StringVar(&screenshotFormat, "format", "jpeg", "Image format: jpeg | png")
	screenshotCmd.Flags().IntVar(&screenshotQuality, "quality", observe.DefaultJPEGQuality, "JPEG quality 1-100")

	observeCmd.Flags().IntVar(&observeMaxWidth, "width", 0, "Max screenshot preview width in pixels (0 = original size; does not change tap coordinates)")
	observeCmd.Flags().StringVar(&observeFormat, "format", "jpeg", "Screenshot format: jpeg | png")
	observeCmd.Flags().IntVar(&observeQuality, "quality", observe.DefaultJPEGQuality, "JPEG quality 1-100")
	observeCmd.Flags().StringVar(&observeFile, "file", "", "Screenshot output path (default: $TMPDIR/adb-claw-observe.jpg)")
	observeCmd.Flags().BoolVar(&observeInline, "inline", false, "Also include base64 screenshot in JSON (large; avoid in agent loops)")

	rootCmd.AddCommand(observeCmd)
	rootCmd.AddCommand(screenshotCmd)
}
