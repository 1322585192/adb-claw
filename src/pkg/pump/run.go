package pump

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/llm-net/adb-claw/pkg/adb"
	"github.com/llm-net/adb-claw/pkg/frame"
	"github.com/llm-net/adb-claw/pkg/stream"
)

// Options configure the background latest-frame pump.
type Options struct {
	Serial   string
	Interval time.Duration
	Width    int
	Quality  int
}

// Status is a point-in-time view of the livestream.
type Status struct {
	Running  bool           `json:"running"`
	PID      int            `json:"pid,omitempty"`
	Serial   string         `json:"serial"`
	Key      string         `json:"key"`
	AgeMs    int64          `json:"age_ms,omitempty"`
	Latest   *stream.Latest `json:"latest,omitempty"`
	JPEGPath string         `json:"jpeg_path"`
	LogPath  string         `json:"log_path"`
}

// Run starts a persistent Frame DEX / screencap source and keeps only the
// newest JPEG. It blocks until ctx is cancelled.
func Run(ctx context.Context, client *adb.Client, opts Options) error {
	if client == nil {
		return fmt.Errorf("pump requires an adb client")
	}
	if opts.Interval <= 0 {
		opts.Interval = 250 * time.Millisecond
	}
	if opts.Quality <= 0 {
		opts.Quality = frame.QualityHigh
	}
	key := stream.Sanitize(opts.Serial)
	if opts.Serial == "" {
		key = stream.ResolveKey(client)
	}
	if err := os.MkdirAll(stream.Dir(key), 0755); err != nil {
		return err
	}
	if err := stream.WritePID(key, os.Getpid()); err != nil {
		return err
	}
	defer stream.RemovePID(key)

	src, err := frame.Start(ctx, client, frame.Options{
		Interval:   opts.Interval,
		Width:      opts.Width,
		Quality:    opts.Quality,
		LatestPath: stream.JPEGPath(key),
		OnFrame: func(f *frame.Frame) {
			if f == nil {
				return
			}
			_ = stream.Write(key, latestFromFrame(f, key, opts))
		},
	})
	if err != nil {
		return err
	}
	defer src.Close()

	<-ctx.Done()
	return nil
}

func latestFromFrame(f *frame.Frame, key string, opts Options) stream.Latest {
	mode := f.Source
	if mode == "dex" {
		mode = "stream"
	}
	captured := f.CapturedAt()
	if captured.IsZero() {
		captured = f.ReceivedAt
	}
	return stream.Latest{
		Seq:           f.Seq,
		Hash:          f.Hash,
		CapturedAt:    captured.UTC(),
		UpdatedAt:     time.Now().UTC(),
		Path:          stream.JPEGPath(key),
		Source:        f.Source,
		Mode:          mode,
		DeviceWidth:   int(f.DeviceWidth),
		DeviceHeight:  int(f.DeviceHeight),
		ImageWidth:    int(f.ImageWidth),
		ImageHeight:   int(f.ImageHeight),
		Rotation:      int(f.Rotation),
		RotationKnown: f.RotationKnown,
		Complete:      true,
		Quality:       opts.Quality,
		Width:         opts.Width,
		Serial:        opts.Serial,
	}
}
