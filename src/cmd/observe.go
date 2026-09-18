package cmd

import (
	"strings"
	"time"

	"github.com/llm-net/adb-claw/pkg/observe"
	"github.com/spf13/cobra"
)

var (
	observeMaxWidth    int
	observeFormat      string
	observeQuality     int
	observeFile        string
	observeCaptureMode string
	observeUIMode      string
	observeCompressed  bool
	observeProfile     bool
	observeSkipImage   bool
)

var observeCmd = &cobra.Command{
	Use:   "observe",
	Short: "Capture screenshot + UI tree",
	Long: `Captures a screenshot and UI element tree in parallel, returning both in a single response.

The screenshot is JPEG-compressed and written to a file. JSON never includes image
bytes or base64 — only the file path and size/scale metadata. UI tree bounds/center
stay in device pixels; never tap using preview-image pixels even if --width scaled
the screenshot.

A successful observe writes a versioned snapshot (data.state_id). Later
tap/long-press/scroll/clear-field --index/--id/--text reuse that snapshot
instead of dumping the UI tree again.`,
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
			MaxWidth:   observeMaxWidth,
			Format:     format,
			Quality:    observeQuality,
			Path:       observeFile,
			Mode:       observe.CaptureMode(observeCaptureMode),
			UIMode:     observe.UIMode(observeUIMode),
			Compressed: observeCompressed,
			Profile:    observeProfile,
			SkipImage:  observeSkipImage,
		})

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
	screenshotOutput      string
	screenshotMaxWidth    int
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
			MaxWidth: screenshotMaxWidth,
			Format:   format,
			Quality:  screenshotQuality,
			Path:     path,
			Mode:     observe.CaptureMode(screenshotCaptureMode),
			Profile:  screenshotProfile,
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
	screenshotCmd.Flags().StringVarP(&screenshotOutput, "file", "f", "", "Screenshot output path (default: $TMPDIR/adb-claw-screenshot.jpg)")
	screenshotCmd.Flags().IntVar(&screenshotMaxWidth, "width", 0, "Max image width in pixels (0 = original size; tap coords stay device pixels)")
	screenshotCmd.Flags().StringVar(&screenshotFormat, "format", "jpeg", "Image format: jpeg | png")
	screenshotCmd.Flags().IntVar(&screenshotQuality, "quality", observe.DefaultJPEGQuality, "JPEG quality 1-100")
	screenshotCmd.Flags().StringVar(&screenshotCaptureMode, "capture", "auto", "Capture mode: auto | stream | pull")
	screenshotCmd.Flags().BoolVar(&screenshotProfile, "profile", false, "Include segmented capture timing")

	observeCmd.Flags().IntVar(&observeMaxWidth, "width", 0, "Max screenshot preview width in pixels (0 = original size; does not change tap coordinates)")
	observeCmd.Flags().StringVar(&observeFormat, "format", "jpeg", "Screenshot format: jpeg | png")
	observeCmd.Flags().IntVar(&observeQuality, "quality", observe.DefaultJPEGQuality, "JPEG quality 1-100")
	observeCmd.Flags().StringVar(&observeFile, "file", "", "Screenshot output path (default: $TMPDIR/adb-claw-observe.jpg)")
	observeCmd.Flags().StringVar(&observeCaptureMode, "capture", "auto", "Capture mode: auto | stream | pull")
	observeCmd.Flags().StringVar(&observeUIMode, "ui-mode", "full", "UI tree profile: full | compact | interactive | realtime")
	observeCmd.Flags().BoolVar(&observeCompressed, "compressed", false, "Use uiautomator dump --compressed")
	observeCmd.Flags().BoolVar(&observeProfile, "profile", false, "Include segmented observe timing")
	observeCmd.Flags().BoolVar(&observeSkipImage, "skip-screenshot", false, "Skip screenshot capture and only dump the UI tree")

	rootCmd.AddCommand(observeCmd)
	rootCmd.AddCommand(screenshotCmd)
}
