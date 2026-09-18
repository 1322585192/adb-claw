package cmd

import (
	"strings"
	"time"

	"github.com/spf13/cobra"
)

var shellCmd = &cobra.Command{
	Use:   "shell <command>",
	Short: "Run a raw adb shell command",
	Long: `Execute a raw command on the device via adb shell.
The command output is captured and returned in the JSON envelope.
Example:
  adb-claw shell "ls /sdcard/"
  adb-claw shell "getprop ro.build.version.release"`,
	Args: cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		start := time.Now()

		// Join all args as a single shell command string
		command := strings.Join(args, " ")
		if reason := forbiddenShellReason(command); reason != "" {
			writer.Fail("shell", "SHELL_FORBIDDEN", reason,
				"Use observe/frame.latest for perception, app current for foreground state, or type for Unicode input", start)
			return nil
		}

		writer.Verbose("shell: %s", command)
		result, err := client.Shell(command)
		if err != nil {
			writer.Fail("shell", "ADB_ERROR", err.Error(), "", start)
			return nil
		}

		writer.Success("shell", map[string]interface{}{
			"stdout":    strings.TrimRight(result.Stdout, "\n"),
			"stderr":    strings.TrimRight(result.Stderr, "\n"),
			"exit_code": result.ExitCode,
		}, start)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(shellCmd)
}

func forbiddenShellReason(command string) string {
	normalized := strings.ToLower(strings.Join(strings.Fields(command), " "))

	if strings.Contains(normalized, "uiautomator") &&
		strings.Contains(normalized, "dump") {
		return "UI hierarchy dumps are disabled in image-only mode"
	}
	if strings.Contains(normalized, "dumpsys accessibility") ||
		strings.Contains(normalized, "dumpsys window") ||
		strings.Contains(normalized, "dumpsys activity top") ||
		strings.Contains(normalized, "dumpsys activity activities") {
		return "layout and accessibility probing through dumpsys is disabled in image-only mode"
	}
	if strings.Contains(normalized, "service call clipboard") {
		return "raw clipboard service calls are disabled; adb-claw type handles Unicode"
	}
	if containsIMEChange(normalized) {
		return "changing the device input method is disabled; adb-claw type handles Unicode without an IME"
	}
	if strings.Contains(normalized, "settings put secure default_input_method") ||
		strings.Contains(normalized, "settings put secure enabled_input_methods") {
		return "changing secure input-method settings is disabled"
	}
	return ""
}

func containsIMEChange(command string) bool {
	fields := strings.Fields(command)
	for i, field := range fields {
		if field != "ime" || i+1 >= len(fields) {
			continue
		}
		switch fields[i+1] {
		case "set", "enable", "disable", "reset":
			return true
		}
	}
	return false
}
