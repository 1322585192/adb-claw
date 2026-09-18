package cmd

import (
	"context"
	"encoding/json"
	"os"
	"time"

	"github.com/llm-net/adb-claw/pkg/monitor"
	"github.com/llm-net/adb-claw/pkg/perf"
	"github.com/spf13/cobra"
)

var (
	monitorDuration int
	monitorInterval int
	monitorStream   bool
	monitorProfile  bool
)

var monitorCmd = &cobra.Command{
	Use:   "monitor",
	Short: "Continuously monitor UI text via accessibility framework",
	Long: `Monitor UI text on the device by connecting directly to the Android accessibility
framework. Unlike 'ui tree' which uses uiautomator dump, this command skips video
surface nodes and works reliably during live streams and video playback.

Two modes:
  Bounded (default): runs for --duration ms, returns all captured text in JSON envelope
  Streaming (--stream): outputs each new text as a JSON line in real time

The host owns --duration: the device process is stopped when the deadline is reached
instead of waiting for extra Java sleep/teardown.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		start := time.Now()

		writer.Verbose("monitor: duration=%dms interval=%dms stream=%v", monitorDuration, monitorInterval, monitorStream)

		clock := perf.Start()
		if err := monitor.EnsureDEX(client); err != nil {
			writer.Fail("monitor", "DEX_PUSH_FAILED", err.Error(),
				"Check device connection and try again", start)
			return nil
		}
		ensureMs := clock.Lap()

		if monitorStream {
			return runStreamingMode(start, ensureMs)
		}
		return runBoundedMode(start, ensureMs)
	},
}

func runBoundedMode(start time.Time, ensureMs int64) error {
	count := monitorDuration / monitorInterval
	if count < 1 {
		count = 1
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	clock := perf.Start()
	proc, err := monitor.Start(ctx, client, monitorInterval, count)
	if err != nil {
		writer.Fail("monitor", "MONITOR_START_FAILED", err.Error(),
			"Ensure device is connected and adb is working", start)
		return nil
	}
	startMs := clock.Lap()

	deadline := time.NewTimer(time.Duration(monitorDuration) * time.Millisecond)
	defer deadline.Stop()

	var texts []monitor.TextEntry
	var firstEventMs int64
	collect := func(line string) {
		entry, err := monitor.ParseLine(line)
		if err != nil {
			writer.Verbose("monitor: skip unparseable line: %s", line)
			return
		}
		if firstEventMs == 0 {
			firstEventMs = time.Since(start).Milliseconds()
		}
		texts = append(texts, *entry)
	}
	collecting := true
	for collecting {
		select {
		case line, ok := <-proc.Lines():
			if !ok {
				collecting = false
				break
			}
			collect(line)
		case <-deadline.C:
			proc.Stop()
			for line := range proc.Lines() {
				collect(line)
			}
			collecting = false
		}
	}

	if err := proc.Wait(); err != nil {
		writer.Verbose("monitor: process exit: %v", err)
	}

	if texts == nil {
		texts = []monitor.TextEntry{}
	}

	data := map[string]interface{}{
		"texts":       texts,
		"count":       len(texts),
		"duration_ms": time.Since(start).Milliseconds(),
	}
	if monitorProfile {
		data["profile"] = monitor.Timing{
			EnsureDEXMs:  ensureMs,
			StartMs:      startMs,
			FirstEventMs: firstEventMs,
			ExitMs:       time.Since(start).Milliseconds(),
			TotalMs:      time.Since(start).Milliseconds(),
		}
	}
	writer.Success("monitor", data, start)
	return nil
}

func runStreamingMode(start time.Time, ensureMs int64) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	count := monitorDuration / monitorInterval
	if count < 1 {
		count = 1
	}

	proc, err := monitor.Start(ctx, client, monitorInterval, count)
	if err != nil {
		writer.Fail("monitor", "MONITOR_START_FAILED", err.Error(),
			"Ensure device is connected and adb is working", start)
		return nil
	}

	deadline := time.NewTimer(time.Duration(monitorDuration) * time.Millisecond)
	defer deadline.Stop()

	enc := json.NewEncoder(os.Stdout)
	emit := func(line string) {
		entry, err := monitor.ParseLine(line)
		if err != nil {
			writer.Verbose("monitor: skip unparseable line: %s", line)
			return
		}
		enc.Encode(entry)
	}
	collecting := true
	for collecting {
		select {
		case line, ok := <-proc.Lines():
			if !ok {
				collecting = false
				break
			}
			emit(line)
		case <-deadline.C:
			proc.Stop()
			for line := range proc.Lines() {
				emit(line)
			}
			collecting = false
		}
	}

	if err := proc.Wait(); err != nil {
		writer.Verbose("monitor: process exit: %v", err)
	}
	_ = ensureMs
	return nil
}

func init() {
	monitorCmd.Flags().IntVar(&monitorDuration, "duration", 10000, "Total monitoring duration in milliseconds")
	monitorCmd.Flags().IntVar(&monitorInterval, "interval", 2000, "Poll interval in milliseconds")
	monitorCmd.Flags().BoolVar(&monitorStream, "stream", false, "Streaming mode: output JSON lines instead of envelope")
	monitorCmd.Flags().BoolVar(&monitorProfile, "profile", false, "Include segmented monitor timing")

	rootCmd.AddCommand(monitorCmd)
}
