package cmd

import (
	"strings"
	"time"

	"github.com/llm-net/adb-claw/pkg/frameartifact"
	"github.com/llm-net/adb-claw/pkg/observe"
	"github.com/spf13/cobra"
)

var (
	waitActivity   string
	waitChanged    bool
	waitGone       bool
	waitTimeout    int
	waitInterval   int
	waitAfterFrame string
)

var waitCmd = &cobra.Command{
	Use:   "wait",
	Short: "Wait for an activity or a visual screen change",
	Long: `Wait for a condition on the device.
Examples:
  adb-claw wait --activity .MainActivity
  adb-claw wait --changed --after-frame FRAME_TOKEN
  adb-claw wait --activity .Splash --gone`,
	RunE: func(cmd *cobra.Command, args []string) error {
		start := time.Now()
		if waitActivity == "" && !waitChanged {
			writer.Fail("wait", "MISSING_ARGS",
				"Specify --activity or --changed",
				"Example: adb-claw wait --changed", start)
			return nil
		}
		if waitActivity != "" && waitChanged {
			writer.Fail("wait", "INVALID_ARGS",
				"Use exactly one condition: --activity or --changed",
				"Run separate waits when both conditions matter", start)
			return nil
		}
		if waitAfterFrame != "" && !waitChanged {
			writer.Fail("wait", "INVALID_ARGS",
				"--after-frame requires --changed", "", start)
			return nil
		}
		if waitGone && waitChanged {
			writer.Fail("wait", "INVALID_ARGS",
				"--gone applies only to --activity", "", start)
			return nil
		}

		timeout := time.Duration(waitTimeout) * time.Millisecond
		interval := time.Duration(waitInterval) * time.Millisecond
		if waitChanged {
			if waitAfterFrame == "" {
				writer.Fail("wait", "MISSING_FRAME",
					"--changed requires --after-frame from the screen seen before the action",
					"Run observe and pass its frame_token to --after-frame", start)
				return nil
			}
			baseline, err := frameartifact.Load(waitAfterFrame)
			if err != nil {
				writer.Fail("wait", "STALE_FRAME", err.Error(),
					"Run observe and pass its frame_token to --after-frame", start)
				return nil
			}
			result, err := observe.WaitForChange(client, baseline, timeout, interval)
			if err != nil {
				writer.Fail("wait", "SCREENSHOT_FAILED", err.Error(), "", start)
				return nil
			}
			writer.SuccessCompact("wait", map[string]interface{}{
				"condition":   "changed",
				"changed":     result.Changed,
				"attempts":    result.Attempts,
				"after_frame": baseline.Token,
				"screenshot":  result.Screenshot,
				"next":        "read_path_directly",
			}, start)
			return nil
		}

		deadline := time.Now().Add(timeout)
		attempts := 0

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
	waitCmd.Flags().StringVar(&waitAfterFrame, "after-frame", "", "Baseline frame token for --changed")
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
