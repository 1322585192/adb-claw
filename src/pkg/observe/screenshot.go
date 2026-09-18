package observe

import (
	"bytes"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/llm-net/adb-claw/pkg/adb"
	"github.com/llm-net/adb-claw/pkg/atomicfile"
	"github.com/llm-net/adb-claw/pkg/frameartifact"
	"github.com/llm-net/adb-claw/pkg/input"
	"github.com/llm-net/adb-claw/pkg/perf"

	"golang.org/x/image/draw"
)

// DefaultJPEGQuality is the default JPEG encoding quality (1-100).
const DefaultJPEGQuality = 60

// DefaultObserveWidth is 0: emit the full logical display at its native
// aspect ratio. --width N is an optional uniform downscale cap, not a
// fixed output size such as 720×1560.
const DefaultObserveWidth = 0

// CaptureOptions controls screenshot encoding and how it is returned.
// User-facing screenshots are written to a unique file; internal transient
// captures may return bytes only. JSON never includes image bytes or base64.
type CaptureOptions struct {
	MaxWidth  int         // 0 = original device resolution (preview only; tap coords stay device pixels)
	MaxPixels int         // 0 = unlimited; rotation-invariant encoded pixel budget
	Format    string      // "jpeg" (default) or "png"
	Quality   int         // JPEG quality 1-100; default 70
	Path      string      // write encoded image here; empty → DefaultScreenshotPath(format)
	Mode      CaptureMode // stream | pull | auto
	Profile   bool        // include segmented timing
	Transient bool        // internal capture: return bytes without file/token persistence
}

// ScreenshotResult holds screenshot metadata. ImageWidth/Height describe the
// encoded preview. DeviceWidth/Height are the live screen pixels used to map
// the 0–999 normalized action grid.
type ScreenshotResult struct {
	Format        string         `json:"format"`
	Path          string         `json:"path"`
	FrameToken    string         `json:"frame_token"`
	Hash          string         `json:"hash"`
	CapturedAt    string         `json:"captured_at"`
	Size          int            `json:"size_bytes"`
	DeviceWidth   int            `json:"device_width"`
	DeviceHeight  int            `json:"device_height"`
	ActionWidth   int            `json:"action_width"`
	ActionHeight  int            `json:"action_height"`
	ImageWidth    int            `json:"image_width"`
	ImageHeight   int            `json:"image_height"`
	Rotation      int            `json:"rotation"`
	RotationKnown bool           `json:"rotation_known"`
	Scale         float64        `json:"scale"`
	DisplayWidth  int            `json:"display_width,omitempty"`
	DisplayHeight int            `json:"display_height,omitempty"`
	Complete      bool           `json:"complete"`
	Mode          string         `json:"mode,omitempty"`
	Profile       *TimingProfile `json:"profile,omitempty"`
	Bytes         []byte         `json:"-"`
}

// ObserveOptions controls a screenshot-only observe call.
type ObserveOptions struct {
	MaxWidth  int
	MaxPixels int
	Format    string
	Quality   int
	Path      string
	Mode      CaptureMode
	Profile   bool
}

// DefaultObservePath returns the default on-disk path for an observe screenshot.
func DefaultObservePath(format string) string {
	return frameartifact.Path(frameartifact.NewToken(), format)
}

// DefaultScreenshotPath returns the default on-disk path for the screenshot command.
func DefaultScreenshotPath(format string) string {
	return frameartifact.Path(frameartifact.NewToken(), format)
}

func normalizeCaptureOptions(opts CaptureOptions) (CaptureOptions, error) {
	if opts.Format == "" {
		opts.Format = "jpeg"
	}
	opts.Format = strings.ToLower(opts.Format)
	if opts.Format != "jpeg" && opts.Format != "png" {
		return opts, fmt.Errorf("unsupported screenshot format %q (use jpeg or png)", opts.Format)
	}
	if opts.Quality <= 0 {
		opts.Quality = DefaultJPEGQuality
	}
	if opts.Quality > 100 {
		opts.Quality = 100
	}
	if opts.Path == "" && !opts.Transient {
		opts.Path = DefaultScreenshotPath(opts.Format)
	}
	return opts, nil
}

// CaptureScreenshot captures the device screen, optionally downscales the preview,
// encodes JPEG/PNG, and writes a file. Device pixel size is always reported
// from the original screencap so tap coordinates stay in device space.
func CaptureScreenshot(cmd adb.Commander, opts CaptureOptions) (*ScreenshotResult, error) {
	opts, err := normalizeCaptureOptions(opts)
	if err != nil {
		return nil, err
	}

	clock := perf.Start()
	capturedAt := time.Now().UTC()
	profile := &TimingProfile{}
	mode := resolveCaptureMode(cmd, opts.Mode)
	profile.Mode = string(mode)

	raw, err := capturePNG(cmd, mode, profile)
	if err != nil {
		return nil, err
	}
	src, err := decodePNG(raw)
	profile.DecodeMs = clock.Lap()
	if err != nil {
		return nil, err
	}

	displayW, displayH, haveDisplay := logicalDisplaySize(cmd)
	src, raw, mode = recaptureIfIncomplete(cmd, src, raw, mode, displayW, displayH, haveDisplay, profile)
	srcBounds := src.Bounds()
	deviceW := srcBounds.Dx()
	deviceH := srcBounds.Dy()
	complete := !haveDisplay || captureComplete(deviceW, deviceH, displayW, displayH)
	if complete && haveDisplay {
		deviceW, deviceH = matchDisplayOrientation(deviceW, deviceH, displayW, displayH)
	}
	rotation, rotationErr := input.CurrentRotation(cmd)

	outImg := src
	newW, newH := scaledDimensions(srcBounds.Dx(), srcBounds.Dy(), opts.MaxWidth, opts.MaxPixels)
	if newW != srcBounds.Dx() || newH != srcBounds.Dy() {
		dst := image.NewRGBA(image.Rect(0, 0, newW, newH))
		draw.BiLinear.Scale(dst, dst.Bounds(), src, srcBounds, draw.Over, nil)
		outImg = dst
	}
	profile.ResizeMs = clock.Lap()

	imageW := outImg.Bounds().Dx()
	imageH := outImg.Bounds().Dy()

	data, err := encodeImage(outImg, opts.Format, opts.Quality)
	profile.EncodeMs = clock.Lap()
	if err != nil {
		return nil, err
	}

	if opts.Path != "" {
		if err := atomicfile.Write(opts.Path, data, 0644); err != nil {
			return nil, fmt.Errorf("write screenshot file: %w", err)
		}
	}
	profile.WriteMs = clock.Lap()
	profile.TotalMs = clock.Total()

	token := ""
	if !opts.Transient {
		if pathToken, ok := frameartifact.TokenFromPath(opts.Path); ok {
			token = pathToken
		} else {
			token = frameartifact.NewToken()
		}
	}
	hash := frameartifact.Hash(data)
	result := &ScreenshotResult{
		Format:        opts.Format,
		Path:          opts.Path,
		FrameToken:    token,
		Hash:          hash,
		CapturedAt:    capturedAt.Format(time.RFC3339Nano),
		Size:          len(data),
		DeviceWidth:   deviceW,
		DeviceHeight:  deviceH,
		ActionWidth:   deviceW,
		ActionHeight:  deviceH,
		ImageWidth:    imageW,
		ImageHeight:   imageH,
		Rotation:      rotation,
		RotationKnown: rotationErr == nil,
		Scale:         scaleFactor(imageW, deviceW),
		DisplayWidth:  displayW,
		DisplayHeight: displayH,
		Complete:      complete,
		Mode:          string(mode),
		Bytes:         data,
	}
	if opts.Profile {
		result.Profile = profile
	}
	if !opts.Transient {
		if err := frameartifact.Save(frameartifact.Metadata{
			Token:         token,
			Hash:          hash,
			CapturedAt:    capturedAt,
			Path:          opts.Path,
			Format:        opts.Format,
			CaptureMode:   string(mode),
			Quality:       opts.Quality,
			MaxWidth:      opts.MaxWidth,
			MaxPixels:     opts.MaxPixels,
			DeviceWidth:   deviceW,
			DeviceHeight:  deviceH,
			ActionWidth:   deviceW,
			ActionHeight:  deviceH,
			ImageWidth:    imageW,
			ImageHeight:   imageH,
			Rotation:      rotation,
			RotationKnown: rotationErr == nil,
			Complete:      complete,
		}); err != nil {
			return nil, fmt.Errorf("write frame metadata: %w", err)
		}
	}
	return result, nil
}

func decodePNG(raw []byte) (image.Image, error) {
	if len(raw) == 0 {
		return nil, fmt.Errorf("screencap returned empty data")
	}
	if len(raw) < 8 || string(raw[1:4]) != "PNG" {
		return nil, fmt.Errorf("screencap returned invalid PNG data (%d bytes)", len(raw))
	}
	src, err := png.Decode(bytes.NewReader(raw))
	if err != nil {
		return nil, fmt.Errorf("decode screencap: %w", err)
	}
	return src, nil
}

func logicalDisplaySize(cmd adb.Commander) (int, int, bool) {
	w, h, err := input.GetScreenSize(cmd)
	if err != nil || w <= 0 || h <= 0 {
		return 0, 0, false
	}
	return w, h, true
}

func recaptureIfIncomplete(cmd adb.Commander, src image.Image, raw []byte, mode CaptureMode, displayW, displayH int, haveDisplay bool, profile *TimingProfile) (image.Image, []byte, CaptureMode) {
	if !haveDisplay || captureComplete(src.Bounds().Dx(), src.Bounds().Dy(), displayW, displayH) {
		return src, raw, mode
	}

	attempts := []struct {
		mode CaptureMode
		run  func() ([]byte, error)
	}{
		{CaptureModePull, func() ([]byte, error) { return capturePNG(cmd, CaptureModePull, profile) }},
		{CaptureModeStream, func() ([]byte, error) { return cmd.ExecOut("screencap", "-d", "0", "-p") }},
		{CaptureModeStream, func() ([]byte, error) { return cmd.ExecOut("screencap", "-p") }},
	}
	bestSrc, bestRaw, bestMode := src, raw, mode
	bestArea := src.Bounds().Dx() * src.Bounds().Dy()
	for _, attempt := range attempts {
		nextRaw, err := attempt.run()
		if err != nil {
			continue
		}
		nextSrc, err := decodePNG(nextRaw)
		if err != nil {
			continue
		}
		if captureComplete(nextSrc.Bounds().Dx(), nextSrc.Bounds().Dy(), displayW, displayH) {
			return nextSrc, nextRaw, attempt.mode
		}
		if area := nextSrc.Bounds().Dx() * nextSrc.Bounds().Dy(); area > bestArea {
			bestSrc, bestRaw, bestMode, bestArea = nextSrc, nextRaw, attempt.mode, area
		}
	}
	return bestSrc, bestRaw, bestMode
}

func captureComplete(capW, capH, dispW, dispH int) bool {
	if dispW <= 0 || dispH <= 0 || capW <= 0 || capH <= 0 {
		return true
	}
	return sameAspectSize(capW, capH, dispW, dispH) || sameAspectSize(capW, capH, dispH, dispW)
}

func sameAspectSize(capW, capH, dispW, dispH int) bool {
	if capW == dispW && capH == dispH {
		return true
	}
	wantH := dispH * capW / dispW
	delta := capH - wantH
	if delta < 0 {
		delta = -delta
	}
	tolerance := 2 + wantH/50
	return delta <= tolerance
}

func matchDisplayOrientation(capW, capH, dispW, dispH int) (int, int) {
	if sameAspectSize(capW, capH, dispH, dispW) && !sameAspectSize(capW, capH, dispW, dispH) {
		return dispH, dispW
	}
	return dispW, dispH
}

func capturePNG(cmd adb.Commander, mode CaptureMode, profile *TimingProfile) ([]byte, error) {
	if mode == CaptureModePull {
		return capturePNGPull(cmd, profile)
	}
	clock := perf.Start()
	raw, err := cmd.ExecOut("screencap", "-p")
	profile.ADBCalls++
	profile.TransferMs = clock.Total()
	if err != nil {
		return nil, fmt.Errorf("screencap failed: %w", err)
	}
	return raw, nil
}

func capturePNGPull(cmd adb.Commander, profile *TimingProfile) (data []byte, err error) {
	devicePath := fmt.Sprintf("/data/local/tmp/adbclaw-screencap-%d.png", time.Now().UnixNano())
	localPath := filepath.Join(os.TempDir(), fmt.Sprintf("adbclaw-screencap-%d.png", time.Now().UnixNano()))
	defer os.Remove(localPath)
	defer func() {
		_, _ = cmd.Shell("rm", "-f", devicePath)
		profile.ADBCalls++
	}()

	clock := perf.Start()
	result, err := cmd.Shell("screencap", "-p", devicePath)
	profile.ADBCalls++
	profile.CaptureMs = clock.Lap()
	if err != nil {
		return nil, fmt.Errorf("screencap failed: %w", err)
	}
	if result.ExitCode != 0 {
		msg := strings.TrimSpace(result.Stderr + result.Stdout)
		if msg == "" {
			msg = fmt.Sprintf("screencap exit %d", result.ExitCode)
		}
		return nil, fmt.Errorf("screencap failed: %s", msg)
	}

	pulled, err := cmd.RawCommand("pull", devicePath, localPath)
	profile.ADBCalls++
	profile.TransferMs = clock.Lap()
	if err != nil {
		return nil, fmt.Errorf("pull screenshot failed: %w", err)
	}
	if pulled.ExitCode != 0 {
		msg := strings.TrimSpace(pulled.Stderr + pulled.Stdout)
		if msg == "" {
			msg = fmt.Sprintf("adb pull exit %d", pulled.ExitCode)
		}
		return nil, fmt.Errorf("pull screenshot failed: %s", msg)
	}

	raw, err := os.ReadFile(localPath)
	if err != nil {
		return nil, fmt.Errorf("read pulled screenshot: %w", err)
	}
	return raw, nil
}

func encodeImage(img image.Image, format string, quality int) ([]byte, error) {
	var buf bytes.Buffer
	switch format {
	case "jpeg":
		if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: quality}); err != nil {
			return nil, fmt.Errorf("jpeg encode: %w", err)
		}
	case "png":
		if err := png.Encode(&buf, img); err != nil {
			return nil, fmt.Errorf("png encode: %w", err)
		}
	default:
		return nil, fmt.Errorf("unsupported screenshot format %q", format)
	}
	return buf.Bytes(), nil
}

func formatOrJPEG(format string) string {
	if format == "" {
		return "jpeg"
	}
	return strings.ToLower(format)
}

func scaleFactor(imageWidth, deviceWidth int) float64 {
	if deviceWidth <= 0 {
		return 1
	}
	return float64(imageWidth) / float64(deviceWidth)
}

func scaledDimensions(width, height, maxWidth, maxPixels int) (int, int) {
	if width <= 0 || height <= 0 {
		return width, height
	}
	scale := 1.0
	if maxWidth > 0 && width > maxWidth {
		scale = float64(maxWidth) / float64(width)
	}
	if maxPixels > 0 && int64(width)*int64(height) > int64(maxPixels) {
		pixelScale := math.Sqrt(float64(maxPixels) / float64(int64(width)*int64(height)))
		if pixelScale < scale {
			scale = pixelScale
		}
	}
	if scale >= 1 {
		return width, height
	}
	newW := int(float64(width) * scale)
	newH := int(float64(height) * scale)
	if newW < 1 {
		newW = 1
	}
	if newH < 1 {
		newH = 1
	}
	return newW, newH
}

// TakeScreenshot captures the device screen via "adb exec-out screencap -p".
// Returns encoded PNG bytes. If maxWidth > 0, the preview is downscaled.
// Prefer CaptureScreenshot for new code.
func TakeScreenshot(cmd adb.Commander, maxWidth int) ([]byte, error) {
	result, err := CaptureScreenshot(cmd, CaptureOptions{
		MaxWidth:  maxWidth,
		Format:    "png",
		Mode:      CaptureModeStream,
		Transient: true,
	})
	if err != nil {
		return nil, err
	}
	return result.Bytes, nil
}
