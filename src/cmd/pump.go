package cmd

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/llm-net/adb-claw/pkg/pump"
	"github.com/spf13/cobra"
)

var (
	pumpIntervalMs int
	pumpWidth      int
	pumpQuality    int
	pumpDaemon     bool
)

var pumpCmd = &cobra.Command{
	Use:   "pump",
	Short: "Keep a latest-frame livestream for one-shot observe",
	Long: `Runs a persistent JPEG frame source and keeps only the newest frame.

observe and wait --changed attach to this process: they copy latest.jpg to a
unique token path instead of restarting screencap. The first observe starts
the pump automatically; this command is for explicit start/status/stop.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		start := time.Now()
		opts := pump.Options{
			Serial:   flagSerial,
			Interval: time.Duration(pumpIntervalMs) * time.Millisecond,
			Width:    pumpWidth,
			Quality:  pumpQuality,
		}
		if pumpDaemon {
			if err := pump.Ensure(client, opts); err != nil {
				writer.Fail("pump", "PUMP_FAILED", err.Error(),
					"Check device connection: adb-claw device list", start)
				return nil
			}
			status := pump.Inspect(client)
			writer.Success("pump", status, start)
			return nil
		}

		ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer cancel()
		status := pump.Inspect(client)
		writer.Success("pump", map[string]interface{}{
			"running":   true,
			"pid":       os.Getpid(),
			"serial":    flagSerial,
			"key":       status.Key,
			"jpeg_path": status.JPEGPath,
		}, start)
		if err := pump.Run(ctx, client, opts); err != nil && ctx.Err() == nil {
			writer.Fail("pump", "PUMP_FAILED", err.Error(),
				"Check device connection: adb-claw device list", time.Now())
			return nil
		}
		return nil
	},
}

var pumpStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show the latest-frame pump",
	RunE: func(cmd *cobra.Command, args []string) error {
		start := time.Now()
		writer.Success("pump status", pump.Inspect(client), start)
		return nil
	},
}

var pumpStopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop the latest-frame pump",
	RunE: func(cmd *cobra.Command, args []string) error {
		start := time.Now()
		if err := pump.Stop(client); err != nil {
			writer.Fail("pump stop", "PUMP_FAILED", err.Error(), "", start)
			return nil
		}
		writer.Success("pump stop", map[string]bool{"stopped": true}, start)
		return nil
	},
}

func init() {
	pumpCmd.Flags().IntVar(&pumpIntervalMs, "interval", 250, "Capture interval in milliseconds")
	pumpCmd.Flags().IntVar(&pumpWidth, "width", 0, "Optional max encode width (0 = native aspect)")
	pumpCmd.Flags().IntVar(&pumpQuality, "quality", 60, "JPEG quality 1-100")
	pumpCmd.Flags().BoolVar(&pumpDaemon, "daemon", false, "Detach and keep writing latest.jpg")
	pumpCmd.AddCommand(pumpStatusCmd)
	pumpCmd.AddCommand(pumpStopCmd)
	rootCmd.AddCommand(pumpCmd)
}
