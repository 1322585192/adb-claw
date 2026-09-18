package observe

import (
	"bytes"
	"fmt"
	"image"
	"image/jpeg"
	"os"
	"strings"
	"time"

	"github.com/llm-net/adb-claw/pkg/adb"
	"github.com/llm-net/adb-claw/pkg/atomicfile"
	"github.com/llm-net/adb-claw/pkg/frameartifact"
	"github.com/llm-net/adb-claw/pkg/perf"
	"github.com/llm-net/adb-claw/pkg/stream"

	"golang.org/x/image/draw"
)

// EnsureStream starts the background latest-frame pump. The pump package
// registers the real implementation from init so this package does not
// import pump (frame already imports observe).
var EnsureStream func(cmd adb.Commander, opts ObserveOptions) error

const streamFirstWait = 5 * time.Second

func useLiveStream(opts ObserveOptions) bool {
	if opts.DisableStream || os.Getenv(stream.EnvNoPump) != "" {
		return false
	}
	if formatOrJPEG(opts.Format) != "jpeg" {
		return false
	}
	return opts.Mode != CaptureModePull
}

func useLiveStreamMeta(meta *frameartifact.Metadata) bool {
	if os.Getenv(stream.EnvNoPump) != "" {
		return false
	}
	if meta == nil {
		return true
	}
	if strings.EqualFold(meta.Format, "png") {
		return false
	}
	return !strings.EqualFold(meta.CaptureMode, string(CaptureModePull))
}

func observeFromStream(cmd adb.Commander, opts ObserveOptions) (*ScreenshotResult, error) {
	key := stream.ResolveKey(cmd)
	if snap, err := snapshotIfReady(key, opts); err == nil {
		return snap, nil
	}
	if EnsureStream != nil {
		started := time.Now()
		if err := EnsureStream(cmd, opts); err != nil {
			return nil, err
		}
		latest, err := stream.Wait(key, streamFirstWait, func(latest *stream.Latest) bool {
			if latest == nil || latest.Hash == "" {
				return false
			}
			return !latest.UpdatedAt.Before(started)
		})
		if err != nil {
			return nil, err
		}
		return snapshotLatest(latest, opts)
	}
	return nil, fmt.Errorf("no live stream")
}

func snapshotIfReady(key string, opts ObserveOptions) (*ScreenshotResult, error) {
	latest, err := stream.Read(key)
	if err != nil {
		return nil, err
	}
	if latest.Hash == "" {
		return nil, fmt.Errorf("latest frame empty")
	}
	if stream.Running(key) || stream.Age(latest) <= stream.FreshAge {
		return snapshotLatest(latest, opts)
	}
	return nil, fmt.Errorf("latest frame stale")
}

func snapshotLatest(latest *stream.Latest, opts ObserveOptions) (*ScreenshotResult, error) {
	clock := perf.Start()
	profile := &TimingProfile{Mode: captureModeOf(latest)}
	data, err := stream.ReadJPEG(latest)
	profile.TransferMs = clock.Lap()
	if err != nil {
		return nil, err
	}

	imageW, imageH := latest.ImageWidth, latest.ImageHeight
	deviceW, deviceH := latest.DeviceWidth, latest.DeviceHeight
	if deviceW <= 0 {
		deviceW = imageW
	}
	if deviceH <= 0 {
		deviceH = imageH
	}

	needScale := opts.MaxWidth > 0 || opts.MaxPixels > 0
	if needScale && imageW > 0 && imageH > 0 {
		newW, newH := scaledDimensions(imageW, imageH, opts.MaxWidth, opts.MaxPixels)
		if newW != imageW || newH != imageH {
			src, err := jpeg.Decode(bytes.NewReader(data))
			profile.DecodeMs = clock.Lap()
			if err != nil {
				return nil, fmt.Errorf("decode latest jpeg: %w", err)
			}
			dst := image.NewRGBA(image.Rect(0, 0, newW, newH))
			draw.BiLinear.Scale(dst, dst.Bounds(), src, src.Bounds(), draw.Over, nil)
			profile.ResizeMs = clock.Lap()
			quality := opts.Quality
			if quality <= 0 {
				quality = latest.Quality
			}
			if quality <= 0 {
				quality = DefaultJPEGQuality
			}
			encoded, err := encodeImage(dst, "jpeg", quality)
			profile.EncodeMs = clock.Lap()
			if err != nil {
				return nil, err
			}
			data = encoded
			imageW, imageH = newW, newH
		}
	}

	path := opts.Path
	if path == "" {
		path = DefaultObservePath("jpeg")
	}
	if err := atomicfile.Write(path, data, 0644); err != nil {
		return nil, fmt.Errorf("write screenshot file: %w", err)
	}
	profile.WriteMs = clock.Lap()
	profile.TotalMs = clock.Total()

	token := ""
	if pathToken, ok := frameartifact.TokenFromPath(path); ok {
		token = pathToken
	} else {
		token = frameartifact.NewToken()
	}
	hash := frameartifact.Hash(data)
	capturedAt := latest.CapturedAt
	if capturedAt.IsZero() {
		capturedAt = time.Now().UTC()
	}
	result := &ScreenshotResult{
		Format:        "jpeg",
		Path:          path,
		FrameToken:    token,
		Hash:          hash,
		CapturedAt:    capturedAt.UTC().Format(time.RFC3339Nano),
		Size:          len(data),
		DeviceWidth:   deviceW,
		DeviceHeight:  deviceH,
		ActionWidth:   deviceW,
		ActionHeight:  deviceH,
		ImageWidth:    imageW,
		ImageHeight:   imageH,
		Rotation:      latest.Rotation,
		RotationKnown: latest.RotationKnown,
		Scale:         scaleFactor(imageW, deviceW),
		Complete:      latest.Complete,
		Mode:          captureModeOf(latest),
		Source:        "pump",
		FrameSeq:      latest.Seq,
		FrameAgeMs:    stream.Age(latest).Milliseconds(),
		Bytes:         data,
	}
	if opts.Profile {
		result.Profile = profile
	}
	if err := frameartifact.Save(frameartifact.Metadata{
		Token:         token,
		Hash:          hash,
		CapturedAt:    capturedAt.UTC(),
		Path:          path,
		Format:        "jpeg",
		CaptureMode:   result.Mode,
		Quality:       opts.Quality,
		MaxWidth:      opts.MaxWidth,
		MaxPixels:     opts.MaxPixels,
		DeviceWidth:   deviceW,
		DeviceHeight:  deviceH,
		ActionWidth:   deviceW,
		ActionHeight:  deviceH,
		ImageWidth:    imageW,
		ImageHeight:   imageH,
		Rotation:      latest.Rotation,
		RotationKnown: latest.RotationKnown,
		Complete:      latest.Complete,
	}); err != nil {
		return nil, fmt.Errorf("write frame metadata: %w", err)
	}
	return result, nil
}

func captureModeOf(latest *stream.Latest) string {
	if latest == nil {
		return "stream"
	}
	if latest.Mode != "" {
		return latest.Mode
	}
	if latest.Source == "pull" {
		return "pull"
	}
	return "stream"
}

func waitStreamChange(key string, baseline *frameartifact.Metadata, timeout, interval time.Duration) (*ChangeResult, error) {
	if interval <= 0 {
		interval = 40 * time.Millisecond
	}
	deadline := time.Now().Add(timeout)
	result := &ChangeResult{}
	for {
		result.Attempts++
		latest, err := stream.Read(key)
		if err == nil {
			changed := latest.Hash != baseline.Hash ||
				latest.Rotation != baseline.Rotation ||
				latest.DeviceWidth != baseline.ActionWidth ||
				latest.DeviceHeight != baseline.ActionHeight
			if changed || !time.Now().Before(deadline) {
				snap, snapErr := snapshotLatest(latest, ObserveOptions{
					MaxWidth:  baseline.MaxWidth,
					MaxPixels: baseline.MaxPixels,
					Format:    "jpeg",
					Quality:   baseline.Quality,
				})
				if snapErr != nil {
					return nil, snapErr
				}
				result.Screenshot = snap
				result.Changed = changed
				return result, nil
			}
		}
		remain := time.Until(deadline)
		if remain <= 0 {
			if result.Screenshot != nil {
				return result, nil
			}
			if err != nil {
				return nil, err
			}
			return result, fmt.Errorf("latest frame unavailable")
		}
		time.Sleep(minDuration(interval, remain))
	}
}
