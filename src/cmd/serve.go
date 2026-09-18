package cmd

import (
	"os"
	"time"

	"github.com/llm-net/adb-claw/pkg/frame"
	"github.com/llm-net/adb-claw/pkg/server"
	"github.com/spf13/cobra"
)

var (
	serveStdio     bool
	serveWidth     int
	serveQuality   int
	serveForceHigh bool
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Args:  cobra.NoArgs,
	Short: "Persistent JSONL frame + normalized-action session",
	Long: `Reads JSONL requests from stdin and writes JSONL responses to stdout.

Methods:
  ping
  frame.latest
  frame.wait_after  {frame_seq, timeout_ms}
  act               {frame_seq, action, x, y, ...}   // 0-999 normalized
  device.info
  close

frame.latest writes latest.jpg and returns frame_seq. act maps 0-999 onto
the device pixels of that frame. A stale or changed frame returns STALE_FRAME.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		_ = serveStdio
		srv := &server.Server{
			Cmd:    client,
			Client: client,
			Serial: flagSerial,
			In:     os.Stdin,
			Out:    os.Stdout,
			Options: server.Options{
				Width:     serveWidth,
				Quality:   serveQuality,
				MaxAge:    server.DefaultMaxFrameAge,
				ForceHigh: serveForceHigh,
			},
		}
		if err := srv.Run(); err != nil {
			writer.Fail("serve", "SERVE_FAILED", err.Error(), "", time.Now())
			return nil
		}
		return nil
	},
}

func init() {
	serveCmd.Flags().BoolVar(&serveStdio, "stdio", true, "Read JSONL from stdin and write JSONL to stdout")
	serveCmd.Flags().IntVar(&serveWidth, "width", 0, "Optional max frame width (0 = native aspect)")
	serveCmd.Flags().IntVar(&serveQuality, "quality", frame.QualityHigh, "Initial JPEG quality")
	serveCmd.Flags().BoolVar(&serveForceHigh, "resolution-720", false, "Force a new session to start at width 720")
	rootCmd.AddCommand(serveCmd)
}
