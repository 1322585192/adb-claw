package observe

import "github.com/llm-net/adb-claw/pkg/adb"

// CaptureMode selects how a screenshot is fetched from the device.
type CaptureMode string

const (
	CaptureModeAuto   CaptureMode = "auto"
	CaptureModeStream CaptureMode = "stream"
	CaptureModePull   CaptureMode = "pull"
)

// UIMode selects how much UI tree data is returned.
type UIMode string

const (
	UIModeFull        UIMode = "full"
	UIModeCompact     UIMode = "compact"
	UIModeInteractive UIMode = "interactive"
	UIModeRealtime    UIMode = "realtime"
)

// TimingProfile records segmented latency for one observe/screenshot/dump call.
type TimingProfile struct {
	Mode       string `json:"mode,omitempty"`
	ADBCalls   int    `json:"adb_calls,omitempty"`
	CaptureMs  int64  `json:"capture_ms,omitempty"`
	TransferMs int64  `json:"transfer_ms,omitempty"`
	DecodeMs   int64  `json:"decode_ms,omitempty"`
	ResizeMs   int64  `json:"resize_ms,omitempty"`
	EncodeMs   int64  `json:"encode_ms,omitempty"`
	WriteMs    int64  `json:"write_ms,omitempty"`
	DumpMs     int64  `json:"dump_ms,omitempty"`
	ParseMs    int64  `json:"parse_ms,omitempty"`
	TotalMs    int64  `json:"total_ms,omitempty"`
}

// ObserveProfile is the combined profile for a parallel observe call.
type ObserveProfile struct {
	Screenshot *TimingProfile `json:"screenshot,omitempty"`
	UI         *TimingProfile `json:"ui,omitempty"`
	TotalMs    int64          `json:"total_ms,omitempty"`
}

func commanderSerial(cmd adb.Commander) string {
	if c, ok := cmd.(*adb.Client); ok {
		return c.Serial
	}
	return ""
}

func resolveCaptureMode(cmd adb.Commander, mode CaptureMode) CaptureMode {
	switch mode {
	case CaptureModeStream, CaptureModePull:
		return mode
	default:
		if c, ok := cmd.(*adb.Client); ok && looksRemoteSerial(c.Serial) {
			return CaptureModePull
		}
		return CaptureModeStream
	}
}

func looksRemoteSerial(serial string) bool {
	for i := 0; i < len(serial); i++ {
		if serial[i] == ':' {
			return true
		}
	}
	return false
}

func normalizeUIMode(mode UIMode) UIMode {
	switch mode {
	case UIModeCompact, UIModeInteractive, UIModeRealtime:
		return mode
	default:
		return UIModeFull
	}
}
