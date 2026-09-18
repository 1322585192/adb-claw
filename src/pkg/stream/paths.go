package stream

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/llm-net/adb-claw/pkg/adb"
)

const (
	// DefaultKey is the stream directory when no device serial is known.
	DefaultKey = "default"
	// EnvNoPump disables the background livestream for one process.
	EnvNoPump = "ADB_CLAW_NOPUMP"
)

// Dir is the per-device livestream directory under $TMPDIR.
func Dir(key string) string {
	return filepath.Join(os.TempDir(), "adb-claw", "stream", Sanitize(key))
}

// JPEGPath is the capacity-1 latest JPEG written by the pump.
func JPEGPath(key string) string {
	return filepath.Join(Dir(key), "latest.jpg")
}

// JSONPath is the sidecar for the latest JPEG.
func JSONPath(key string) string {
	return filepath.Join(Dir(key), "latest.json")
}

// PIDPath records the detached pump process.
func PIDPath(key string) string {
	return filepath.Join(Dir(key), "pump.pid")
}

// LogPath is the detached pump log.
func LogPath(key string) string {
	return filepath.Join(Dir(key), "pump.log")
}

// LockPath is an exclusive lock used while starting or stopping the pump.
func LockPath(key string) string {
	return filepath.Join(Dir(key), "pump.lock")
}

// Sanitize turns an ADB serial into a single path component.
func Sanitize(serial string) string {
	serial = strings.TrimSpace(serial)
	if serial == "" {
		return DefaultKey
	}
	var b strings.Builder
	for _, r := range serial {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '-', r == '_':
			b.WriteRune(r)
		default:
			b.WriteByte('_')
		}
	}
	if b.Len() == 0 {
		return DefaultKey
	}
	return b.String()
}

// ResolveKey maps an ADB client onto a stable stream directory name.
func ResolveKey(cmd adb.Commander) string {
	client, ok := cmd.(*adb.Client)
	if !ok || client == nil {
		return DefaultKey
	}
	if client.Serial != "" {
		return Sanitize(client.Serial)
	}
	result, err := client.RawCommand("get-serialno")
	if err != nil || result == nil {
		return DefaultKey
	}
	id := strings.TrimSpace(result.Stdout)
	if id == "" || strings.EqualFold(id, "unknown") {
		return DefaultKey
	}
	return Sanitize(id)
}

// ClientSerial is the raw -s value, or empty for the default device.
func ClientSerial(cmd adb.Commander) string {
	if client, ok := cmd.(*adb.Client); ok && client != nil {
		return client.Serial
	}
	return ""
}
