package observe

import (
	"fmt"
	"time"

	"github.com/llm-net/adb-claw/pkg/atomicfile"
	"github.com/llm-net/adb-claw/pkg/frameartifact"
)

func persistWaitFrame(frame *ScreenshotResult) error {
	if frame == nil {
		return fmt.Errorf("no screenshot")
	}
	if frame.Path != "" && frame.FrameToken != "" {
		return nil
	}
	if len(frame.Bytes) == 0 {
		return fmt.Errorf("screenshot has no bytes")
	}
	format := frame.Format
	if format == "" {
		format = "jpeg"
	}
	path := DefaultObservePath(format)
	if err := atomicfile.Write(path, frame.Bytes, 0644); err != nil {
		return fmt.Errorf("write screenshot file: %w", err)
	}
	token, ok := frameartifact.TokenFromPath(path)
	if !ok {
		token = frameartifact.NewToken()
	}
	if err := frameartifact.Save(waitFrameMeta(frame, token, path, format)); err != nil {
		return err
	}
	frame.Path = path
	frame.FrameToken = token
	return nil
}

func waitFrameMeta(frame *ScreenshotResult, token, path, format string) frameartifact.Metadata {
	capturedAt, err := time.Parse(time.RFC3339Nano, frame.CapturedAt)
	if err != nil {
		capturedAt = time.Now().UTC()
	}
	return frameartifact.Metadata{
		Token:         token,
		Hash:          frame.Hash,
		CapturedAt:    capturedAt,
		Path:          path,
		Format:        format,
		CaptureMode:   frame.Mode,
		DeviceWidth:   frame.DeviceWidth,
		DeviceHeight:  frame.DeviceHeight,
		ActionWidth:   frame.ActionWidth,
		ActionHeight:  frame.ActionHeight,
		ImageWidth:    frame.ImageWidth,
		ImageHeight:   frame.ImageHeight,
		Rotation:      frame.Rotation,
		RotationKnown: frame.RotationKnown,
		Complete:      frame.Complete,
	}
}
