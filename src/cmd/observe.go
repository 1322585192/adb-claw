package cmd

import (
	"strings"
	"time"

	"github.com/llm-net/adb-claw/pkg/observe"
	"github.com/spf13/cobra"
)

var (
	observeMaxWidth    int
	observeMaxPixels   int
	observeFormat      string
	observeQuality     int
	observeFile        string
	observeCaptureMode string
	observeProfile     bool
)

var observeCmd = &cobra.Command{
	Use:   "observe",
	Short: "Capture a screenshot frame for visual analysis",
	Long: `Captures a JPEG screenshot and writes it to a file.

JSON never includes image bytes or base64 — only the file path and size/scale
metadata. Read the unique data.screenshot.path and retain frame_token. Act with
--normalized 0-999 coordinates plus --frame TOKEN; coordinates map to that
frame's action_width / action_height.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		start := time.Now()
		writer.Verbose("capturing screenshot frame")

		format := strings.ToLower(observeFormat)
		if format != "jpeg" && format != "png" {
			writer.Fail("observe", "INVALID_ARGS",
				"Unsupported screenshot format '"+observeFormat+"'",
				"Use --format jpeg or --format png", start)
			return nil
		}

		result := observe.Observe(client, observe.ObserveOptions{
			MaxWidth:  observeMaxWidth,
			MaxPixels: observeMaxPixels,
			Format:    format,
			Quality:   observeQuality,
			Path:      observeFile,
			Mode:      observe.CaptureMode(observeCaptureMode),
			Profile:   observeProfile,
		})

		if result.Screenshot == nil {
			writer.Fail("observe", "OBSERVE_FAILED",
				"Screenshot failed",
				"Check device connection: adb-claw device list", start)
			return nil
		}

		writer.SuccessCompact("observe", result, start)
		return nil
	},
}

var (
	screenshotOutput      string
	screenshotMaxWidth    int
	screenshotMaxPixels   int
	screenshotFormat      string
	screenshotQuality     int
	screenshotCaptureMode string
	screenshotProfile     bool
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

		path := screenshotOutput
		if path == "" {
			path = observe.DefaultScreenshotPath(format)
		}

		result, err := observe.CaptureScreenshot(client, observe.CaptureOptions{
			MaxWidth:  screenshotMaxWidth,
			MaxPixels: screenshotMaxPixels,
			Format:    format,
			Quality:   screenshotQuality,
			Path:      path,
			Mode:      observe.CaptureMode(screenshotCaptureMode),
			Profile:   screenshotProfile,
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
	screenshotCmd.Flags().StringVarP(&screenshotOutput, "file", "f", "", "Screenshot output path (default: unique temp frame path)")
	screenshotCmd.Flags().IntVar(&screenshotMaxWidth, "width", 0, "Max image width in pixels (0 = original size)")
	screenshotCmd.Flags().IntVar(&screenshotMaxPixels, "max-pixels", 0, "Rotation-invariant encoded pixel budget (0 = native)")
	screenshotCmd.Flags().StringVar(&screenshotFormat, "format", "jpeg", "Image format: jpeg | png")
	screenshotCmd.Flags().IntVar(&screenshotQuality, "quality", observe.DefaultJPEGQuality, "JPEG quality 1-100")
	screenshotCmd.Flags().StringVar(&screenshotCaptureMode, "capture", "auto", "Capture mode: auto | stream | pull")
	screenshotCmd.Flags().BoolVar(&screenshotProfile, "profile", false, "Include segmented capture timing")

	observeCmd.Flags().IntVar(&observeMaxWidth, "width", observe.DefaultObserveWidth, "Max screenshot width (0 = native size, keep device aspect)")
	observeCmd.Flags().IntVar(&observeMaxPixels, "max-pixels", 0, "Rotation-invariant encoded pixel budget (0 = native)")
	observeCmd.Flags().StringVar(&observeFormat, "format", "jpeg", "Screenshot format: jpeg | png")
	observeCmd.Flags().IntVar(&observeQuality, "quality", observe.DefaultJPEGQuality, "JPEG quality 1-100")
	observeCmd.Flags().StringVar(&observeFile, "file", "", "Screenshot output path (default: unique temp frame path)")
	observeCmd.Flags().StringVar(&observeCaptureMode, "capture", "auto", "Capture mode: auto | stream | pull")
	observeCmd.Flags().BoolVar(&observeProfile, "profile", false, "Include segmented observe timing")

	rootCmd.AddCommand(observeCmd)
	rootCmd.AddCommand(screenshotCmd)
}
