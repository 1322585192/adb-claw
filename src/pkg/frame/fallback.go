package frame

import (
	"time"

	"github.com/llm-net/adb-claw/pkg/adb"
	"github.com/llm-net/adb-claw/pkg/observe"
)

func captureFallback(cmd adb.Commander, seq uint32, width, quality int) (*Frame, error) {
	start := time.Now()
	res, err := observe.CaptureScreenshot(cmd, observe.CaptureOptions{
		MaxWidth: width,
		Format:   "jpeg",
		Quality:  quality,
		Mode:     observe.CaptureModeAuto,
	})
	if err != nil {
		return nil, err
	}
	captured := time.Now()
	return &Frame{
		Header: Header{
			Version:      Version,
			Seq:          seq,
			CapturedAtMs: uint64(captured.UnixMilli()),
			DeviceWidth:  uint16(res.DeviceWidth),
			DeviceHeight: uint16(res.DeviceHeight),
			ImageWidth:   uint16(res.ImageWidth),
			ImageHeight:  uint16(res.ImageHeight),
			EncodeMs:     uint16(time.Since(start).Milliseconds()),
			JPEGLength:   uint32(len(res.Bytes)),
		},
		JPEG:       res.Bytes,
		Hash:       HashJPEG(res.Bytes),
		ReceivedAt: captured,
		TransferMs: time.Since(start).Milliseconds(),
		Source:     res.Mode,
	}, nil
}
