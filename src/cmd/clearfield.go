package cmd

import (
	"time"

	"github.com/llm-net/adb-claw/pkg/input"
	"github.com/spf13/cobra"
)

var clearFieldCmd = &cobra.Command{
	Use:   "clear-field",
	Short: "Clear text in the focused input field",
	Long:  `Clear the currently focused input field. Tap the field first using normalized coordinates.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		start := time.Now()
		writer.Verbose("clearing field")
		method, err := input.ClearField(client)
		if err != nil {
			writer.Fail("clear-field", "CLEAR_FAILED", err.Error(), "", start)
			return nil
		}
		writer.Success("clear-field", map[string]interface{}{
			"method": method,
		}, start)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(clearFieldCmd)
}
