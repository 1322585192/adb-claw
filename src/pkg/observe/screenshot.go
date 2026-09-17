package observe

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"os"
	"path/filepath"
	"strings"

	"github.com/llm-net/adb-claw/pkg/adb"

	"golang.org/x/image/draw"
)

// DefaultJPEGQuality is the default JPEG encoding quality (1-100).
const DefaultJPEGQuality = 70

// DefaultObserveFileName is the temp file used when observe does not get --file.
const DefaultObserveFileName = "adb-claw-observe"

// CaptureOptions controls screenshot encoding and how it is returned.
type CaptureOptions struct {
	MaxWidth int    // 0 = original device resolution (preview only; tap coords stay device pixels)
	Format   string // "jpeg" (default) or "png"
	Quality  int    // JPEG quality 1-100; default 70
	Path     string // write encoded image here; empty means no file (observe always supplies a path)
	Inline   bool   // include base64 in the result
}

// ScreenshotResult holds screenshot metadata. Coordinates for tapping live in the UI tree
// (device pixels). ImageWidth/Height describe the encoded preview only.
type ScreenshotResult struct {
	Format       string  `json:"format"`
	Base64       string  `json:"base64,omitempty"`
	Path         string  `json:"path,omitempty"`
	Size         int     `json:"size_bytes"`
	DeviceWidth  int     `json:"device_width"`
	DeviceHeight int     `json:"device_height"`
	ImageWidth   int     `json:"image_width"`
	ImageHeight  int     `json:"image_height"`
	Scale        float64 `json:"scale"`
	Bytes        []byte  `json:"-"`
}

// ObserveOptions controls a combined observe call.
type ObserveOptions struct {
	MaxWidth int
	Format   string
	Quality  int
	Path     string
	Inline   bool
}

// DefaultObservePath returns the default on-disk path for an observe screenshot.
func DefaultObservePath(format string) string {
	ext := "jpg"
	if strings.ToLower(format) == "png" {
		ext = "png"
	}
	return filepath.Join(os.TempDir(), DefaultObserveFileName+"."+ext)
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
	return opts, nil
}

// CaptureScreenshot captures the device screen, optionally downscales the preview,
// encodes JPEG/PNG, and optionally writes a file. Device pixel size is always reported
// from the original screencap so tap coordinates stay in device space.
func CaptureScreenshot(cmd adb.Commander, opts CaptureOptions) (*ScreenshotResult, error) {
	opts, err := normalizeCaptureOptions(opts)
	if err != nil {
		return nil, err
	}

	raw, err := cmd.ExecOut("screencap", "-p")
	if err != nil {
		return nil, fmt.Errorf("screencap failed: %w", err)
	}
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
	srcBounds := src.Bounds()
	deviceW := srcBounds.Dx()
	deviceH := srcBounds.Dy()

	outImg := src
	if opts.MaxWidth > 0 && deviceW > opts.MaxWidth {
		newW := opts.MaxWidth
		newH := deviceH * newW / deviceW
		dst := image.NewRGBA(image.Rect(0, 0, newW, newH))
		draw.BiLinear.Scale(dst, dst.Bounds(), src, srcBounds, draw.Over, nil)
		outImg = dst
	}

	imageW := outImg.Bounds().Dx()
	imageH := outImg.Bounds().Dy()

	data, err := encodeImage(outImg, opts.Format, opts.Quality)
	if err != nil {
		return nil, err
	}

	result := &ScreenshotResult{
		Format:       opts.Format,
		Size:         len(data),
		DeviceWidth:  deviceW,
		DeviceHeight: deviceH,
		ImageWidth:   imageW,
		ImageHeight:  imageH,
		Scale:        scaleFactor(imageW, deviceW),
		Bytes:        data,
	}
	if opts.Path != "" {
		if err := os.WriteFile(opts.Path, data, 0644); err != nil {
			return nil, fmt.Errorf("write screenshot file: %w", err)
		}
		result.Path = opts.Path
	}
	if opts.Inline {
		result.Base64 = base64.StdEncoding.EncodeToString(data)
	}
	return result, nil
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

// TakeScreenshot captures the device screen via "adb exec-out screencap -p".
// Returns encoded PNG bytes. If maxWidth > 0, the preview is downscaled.
// Prefer CaptureScreenshot for new code.
func TakeScreenshot(cmd adb.Commander, maxWidth int) ([]byte, error) {
	result, err := CaptureScreenshot(cmd, CaptureOptions{
		MaxWidth: maxWidth,
		Format:   "png",
		Inline:   true,
	})
	if err != nil {
		return nil, err
	}
	return result.Bytes, nil
}
