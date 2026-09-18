package cmd

import (
	"os"
	"time"

	"github.com/llm-net/adb-claw/pkg/observe"
	"github.com/llm-net/adb-claw/pkg/server"
	"github.com/spf13/cobra"
)

var (
	serveStdio      bool
	serveUIMode     string
	serveCapture    string
	serveCompressed bool
	serveWidth      int
	serveQuality    int
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Args:  cobra.NoArgs,
	Short: "Persistent JSONL control session for low-latency agents",
	Long: `Reads JSONL requests from stdin and writes JSONL responses to stdout.

Methods:
  ping
  observe  {width, quality, capture, ui_mode, compressed, skip_screenshot, profile}
  act      {state_id, action, index|handle|id|text|x,y}
  close

observe returns a state_id. act uses that snapshot and never re-dumps the UI tree.
A stale or missing state_id fails with STALE_STATE instead of clicking the wrong node.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		_ = serveStdio
		srv := &server.Server{
			Cmd:    client,
			Serial: flagSerial,
			In:     os.Stdin,
			Out:    os.Stdout,
			Options: server.Options{
				CaptureMode: observe.CaptureMode(serveCapture),
				UIMode:      observe.UIMode(serveUIMode),
				Compressed:  serveCompressed,
				MaxWidth:    serveWidth,
				Quality:     serveQuality,
				MaxAge:      observe.DefaultSnapshotMaxAge,
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
	serveCmd.Flags().StringVar(&serveUIMode, "ui-mode", "realtime", "Default UI profile: full | compact | interactive | realtime")
	serveCmd.Flags().StringVar(&serveCapture, "capture", "auto", "Default screenshot capture mode")
	serveCmd.Flags().BoolVar(&serveCompressed, "compressed", false, "Default to compressed UI dump")
	serveCmd.Flags().IntVar(&serveWidth, "width", 540, "Default screenshot preview width")
	serveCmd.Flags().IntVar(&serveQuality, "quality", observe.DefaultJPEGQuality, "Default JPEG quality")
	rootCmd.AddCommand(serveCmd)
}
