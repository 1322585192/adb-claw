package cmd

import (
	"crypto/sha1"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/llm-net/adb-claw/pkg/observe"
	"github.com/spf13/cobra"
)

var (
	waitActivity string
	waitChanged  bool
	waitGone     bool
	waitTimeout  int
	waitInterval int
)

var waitCmd = &cobra.Command{
	Use:   "wait",
	Short: "Wait for an activity or a visual screen change",
	Long: `Wait for a condition on the device.
Examples:
  adb-claw wait --activity .MainActivity
  adb-claw wait --changed
  adb-claw wait --activity .Splash --gone`,
	RunE: func(cmd *cobra.Command, args []string) error {
		start := time.Now()
		if waitActivity == "" && !waitChanged {
			writer.Fail("wait", "MISSING_ARGS",
				"Specify --activity or --changed",
				"Example: adb-claw wait --changed", start)
			return nil
		}

		timeout := time.Duration(waitTimeout) * time.Millisecond
		interval := time.Duration(waitInterval) * time.Millisecond
		deadline := time.Now().Add(timeout)
		attempts := 0

		var baseline string
		if waitChanged {
			var err error
			baseline, err = screenHash()
			if err != nil {
				writer.Fail("wait", "SCREENSHOT_FAILED", err.Error(), "", start)
				return nil
			}
		}

		for time.Now().Before(deadline) {
			attempts++
			if waitActivity != "" {
				found, activity, err := checkActivity(waitActivity)
				if err != nil {
					writer.Verbose("activity check error (attempt %d): %v", attempts, err)
				} else if found && !waitGone {
					writer.Success("wait", map[string]interface{}{
						"condition": "activity",
						"activity":  activity,
						"gone":      false,
						"attempts":  attempts,
						"next":      "observe_once",
					}, start)
					return nil
				} else if !found && waitGone {
					writer.Success("wait", map[string]interface{}{
						"condition": "activity",
						"activity":  waitActivity,
						"gone":      true,
						"attempts":  attempts,
						"next":      "observe_once",
					}, start)
					return nil
				}
			}
			if waitChanged {
				h, err := screenHash()
				if err != nil {
					writer.Verbose("hash error (attempt %d): %v", attempts, err)
				} else if h != baseline {
					writer.Success("wait", map[string]interface{}{
						"condition": "changed",
						"attempts":  attempts,
						"next":      "observe_once",
					}, start)
					return nil
				}
			}
			time.Sleep(interval)
		}

		condition := "activity"
		if waitChanged && waitActivity == "" {
			condition = "screen"
		}
		action := "appear"
		if waitGone {
			action = "disappear"
		}
		if waitChanged && waitActivity == "" {
			action = "change"
		}
		writer.Fail("wait", "WAIT_TIMEOUT",
			condition+" did not "+action+" within "+timeout.String(),
			"Observe the current screen now; increase timeout only when the JPEG is visibly still loading", start)
		return nil
	},
}

func init() {
	waitCmd.Flags().StringVar(&waitActivity, "activity", "", "Wait for this activity to be in the foreground")
	waitCmd.Flags().BoolVar(&waitChanged, "changed", false, "Wait until the screenshot hash changes")
	waitCmd.Flags().BoolVar(&waitGone, "gone", false, "Wait for the activity to disappear")
	waitCmd.Flags().IntVar(&waitTimeout, "timeout", 10000, "Timeout in milliseconds")
	waitCmd.Flags().IntVar(&waitInterval, "interval", 250, "Poll interval in milliseconds")
	rootCmd.AddCommand(waitCmd)
}

func checkActivity(activity string) (bool, string, error) {
	result, err := client.Shell("dumpsys", "window", "displays")
	if err != nil {
		return false, "", err
	}
	for _, line := range strings.Split(result.Stdout, "\n") {
		line = strings.TrimSpace(line)
		if strings.Contains(line, "mCurrentFocus") || strings.Contains(line, "mFocusedWindow") || strings.Contains(line, "mFocusedApp") {
			if strings.Contains(line, activity) {
				return true, extractWindowName(line), nil
			}
		}
	}
	return false, "", nil
}

func screenHash() (string, error) {
	path := filepath.Join(os.TempDir(), "adb-claw-wait.jpg")
	res, err := observe.CaptureScreenshot(client, observe.CaptureOptions{
		MaxWidth: 360,
		Format:   "jpeg",
		Quality:  40,
		Path:     path,
		Mode:     observe.CaptureModeAuto,
	})
	if err != nil {
		return "", err
	}
	sum := sha1.Sum(res.Bytes)
	return hex.EncodeToString(sum[:8]), nil
}
