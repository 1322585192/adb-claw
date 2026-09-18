package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/llm-net/adb-claw/pkg/input"
	"github.com/llm-net/adb-claw/pkg/observe"
	"github.com/llm-net/adb-claw/pkg/perf"
	"github.com/spf13/cobra"
)

var (
	benchRounds int
	benchWidth  int
)

var benchCmd = &cobra.Command{
	Use:   "bench",
	Short: "Measure segmented observe/tap latency on a connected device",
	Long: `Runs repeated observe, UI dump, screenshot, and coordinate tap samples.
Prints count/p50/p95/max in a JSON envelope. Requires a connected device.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		start := time.Now()
		if benchRounds < 1 {
			benchRounds = 3
		}
		dir := filepath.Join(os.TempDir(), "adb-claw-bench")
		_ = os.MkdirAll(dir, 0755)

		type stat struct {
			Count int   `json:"count"`
			P50   int64 `json:"p50_ms"`
			P95   int64 `json:"p95_ms"`
			Max   int64 `json:"max_ms"`
		}
		summarize := func(samples []int64) stat {
			return stat{
				Count: len(samples),
				P50:   perf.Percentile(samples, 50),
				P95:   perf.Percentile(samples, 95),
				Max:   perf.Max(samples),
			}
		}

		var observeMs, dumpMs, shotMs, tapMs []int64
		for i := 0; i < benchRounds; i++ {
			path := filepath.Join(dir, fmt.Sprintf("obs-%d.jpg", i))
			t0 := time.Now()
			res := observe.Observe(client, observe.ObserveOptions{
				MaxWidth: benchWidth,
				Format:   "jpeg",
				Quality:  50,
				Path:     path,
				Mode:     "auto",
				UIMode:   observe.UIModeRealtime,
				Profile:  true,
			})
			observeMs = append(observeMs, time.Since(t0).Milliseconds())
			if res.UI != nil && res.UI.Profile != nil {
				dumpMs = append(dumpMs, res.UI.Profile.TotalMs)
			}
			if res.Screenshot != nil && res.Screenshot.Profile != nil {
				shotMs = append(shotMs, res.Screenshot.Profile.TotalMs)
			}

			t1 := time.Now()
			_ = input.Tap(client, 5, 5)
			tapMs = append(tapMs, time.Since(t1).Milliseconds())
		}

		writer.Success("bench", map[string]interface{}{
			"rounds":     benchRounds,
			"observe":    summarize(observeMs),
			"ui_dump":    summarize(dumpMs),
			"screenshot": summarize(shotMs),
			"tap_xy":     summarize(tapMs),
		}, start)
		return nil
	},
}

func init() {
	benchCmd.Flags().IntVar(&benchRounds, "rounds", 5, "Samples per measurement")
	benchCmd.Flags().IntVar(&benchWidth, "width", 540, "Screenshot preview width")
	rootCmd.AddCommand(benchCmd)
}
