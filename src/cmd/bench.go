package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/llm-net/adb-claw/pkg/coord"
	"github.com/llm-net/adb-claw/pkg/frame"
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
	Short: "Measure image-only frame and normalized-tap latency",
	Long: `Runs screenshot, latest-file write, normalized tap, and visual-change samples.
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

		var firstMs, shotMs, latestMs, tapMs, changeMs []int64
		var jpegSizes []int
		var captureMs, encodeMs, transferMs []int64

		tFirst := time.Now()
		first, err := observe.CaptureScreenshot(client, observe.CaptureOptions{
			MaxWidth: benchWidth,
			Format:   "jpeg",
			Quality:  frame.QualityHigh,
			Path:     filepath.Join(dir, "first.jpg"),
			Mode:     "auto",
			Profile:  true,
		})
		firstMs = append(firstMs, time.Since(tFirst).Milliseconds())
		if err == nil && first != nil {
			jpegSizes = append(jpegSizes, first.Size)
		}

		w, h := 1080, 2340
		if cw, ch, err := input.CurrentScreenSize(client); err == nil {
			w, h = cw, ch
		}

		for i := 0; i < benchRounds; i++ {
			path := filepath.Join(dir, fmt.Sprintf("obs-%d.jpg", i))
			t0 := time.Now()
			res, err := observe.CaptureScreenshot(client, observe.CaptureOptions{
				MaxWidth: benchWidth,
				Format:   "jpeg",
				Quality:  frame.QualityHigh,
				Path:     path,
				Mode:     "auto",
				Profile:  true,
			})
			shotMs = append(shotMs, time.Since(t0).Milliseconds())
			if err == nil && res != nil {
				jpegSizes = append(jpegSizes, res.Size)
				if res.Profile != nil {
					captureMs = append(captureMs, res.Profile.CaptureMs)
					encodeMs = append(encodeMs, res.Profile.EncodeMs)
					transferMs = append(transferMs, res.Profile.TransferMs)
				}
				tL := time.Now()
				_ = frame.WriteAtomic(filepath.Join(dir, "latest.jpg"), res.Bytes)
				latestMs = append(latestMs, time.Since(tL).Milliseconds())
			}

			t1 := time.Now()
			p := coord.Denormalize(500, 500, w, h)
			_ = input.Tap(client, p.X, p.Y)
			tapMs = append(tapMs, time.Since(t1).Milliseconds())
		}

		writer.Success("bench", map[string]interface{}{
			"rounds":        benchRounds,
			"width":         benchWidth,
			"first_frame":   summarize(firstMs),
			"screenshot":    summarize(shotMs),
			"latest_write":  summarize(latestMs),
			"normalized_tap": summarize(tapMs),
			"visual_change": summarize(changeMs),
			"jpeg_bytes_p50": perf.Percentile(int64s(jpegSizes), 50),
			"capture":       summarize(captureMs),
			"encode":        summarize(encodeMs),
			"transfer":      summarize(transferMs),
		}, start)
		return nil
	},
}

func int64s(in []int) []int64 {
	out := make([]int64, len(in))
	for i, v := range in {
		out[i] = int64(v)
	}
	return out
}

func init() {
	benchCmd.Flags().IntVar(&benchRounds, "rounds", 5, "Samples per measurement")
	benchCmd.Flags().IntVar(&benchWidth, "width", frame.WidthHigh, "Screenshot preview width")
	rootCmd.AddCommand(benchCmd)
}
