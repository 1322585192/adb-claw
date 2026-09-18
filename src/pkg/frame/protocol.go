package frame

import (
	"crypto/sha1"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io"
	"time"
)

const (
	// Magic is the 4-byte stream identifier "ADBF".
	Magic = 0x41444246
	// Version is the current binary frame protocol version.
	Version = 1
	// HeaderSize is the fixed big-endian header length before JPEG bytes.
	HeaderSize = 32
)

// Header is the fixed binary prefix of one JPEG frame.
type Header struct {
	Version      uint8
	Rotation     uint8
	Seq          uint32
	CapturedAtMs uint64
	DeviceWidth  uint16
	DeviceHeight uint16
	ImageWidth   uint16
	ImageHeight  uint16
	EncodeMs     uint16
	JPEGLength   uint32
}

// Frame is one decoded screen frame plus host-side receive metadata.
type Frame struct {
	Header
	JPEG       []byte
	Hash       string
	ReceivedAt time.Time
	TransferMs int64
	Duplicate  bool
	Source     string
}

// CapturedAt returns the device capture timestamp.
func (f *Frame) CapturedAt() time.Time {
	if f == nil || f.CapturedAtMs == 0 {
		return time.Time{}
	}
	return time.UnixMilli(int64(f.CapturedAtMs)).UTC()
}

// Age returns how old the frame is relative to now.
func (f *Frame) Age() time.Duration {
	if f == nil {
		return 0
	}
	if !f.CapturedAt().IsZero() {
		return time.Since(f.CapturedAt())
	}
	if !f.ReceivedAt.IsZero() {
		return time.Since(f.ReceivedAt)
	}
	return 0
}

// Scale is image_width / device_width.
func (f *Frame) Scale() float64 {
	if f == nil || f.DeviceWidth == 0 {
		return 1
	}
	return float64(f.ImageWidth) / float64(f.DeviceWidth)
}

// ResolutionBucket is the encoded image width. Native frames report the
// real pixel width rather than a fixed 720/540 bucket.
func (f *Frame) ResolutionBucket() int {
	if f == nil {
		return 0
	}
	return int(f.ImageWidth)
}

// EncodeHeader writes the 32-byte header into buf.
func EncodeHeader(h Header, buf []byte) {
	if len(buf) < HeaderSize {
		return
	}
	binary.BigEndian.PutUint32(buf[0:4], Magic)
	buf[4] = h.Version
	if buf[4] == 0 {
		buf[4] = Version
	}
	buf[5] = h.Rotation
	binary.BigEndian.PutUint32(buf[6:10], h.Seq)
	binary.BigEndian.PutUint64(buf[10:18], h.CapturedAtMs)
	binary.BigEndian.PutUint16(buf[18:20], h.DeviceWidth)
	binary.BigEndian.PutUint16(buf[20:22], h.DeviceHeight)
	binary.BigEndian.PutUint16(buf[22:24], h.ImageWidth)
	binary.BigEndian.PutUint16(buf[24:26], h.ImageHeight)
	binary.BigEndian.PutUint16(buf[26:28], h.EncodeMs)
	binary.BigEndian.PutUint32(buf[28:32], h.JPEGLength)
}

// DecodeHeader parses and validates a 32-byte header.
func DecodeHeader(buf []byte) (Header, error) {
	var h Header
	if len(buf) < HeaderSize {
		return h, fmt.Errorf("short frame header: %d bytes", len(buf))
	}
	magic := binary.BigEndian.Uint32(buf[0:4])
	if magic != Magic {
		return h, fmt.Errorf("bad frame magic 0x%08x", magic)
	}
	h.Version = buf[4]
	if h.Version != Version {
		return h, fmt.Errorf("unsupported frame version %d", h.Version)
	}
	h.Rotation = buf[5]
	h.Seq = binary.BigEndian.Uint32(buf[6:10])
	h.CapturedAtMs = binary.BigEndian.Uint64(buf[10:18])
	h.DeviceWidth = binary.BigEndian.Uint16(buf[18:20])
	h.DeviceHeight = binary.BigEndian.Uint16(buf[20:22])
	h.ImageWidth = binary.BigEndian.Uint16(buf[22:24])
	h.ImageHeight = binary.BigEndian.Uint16(buf[24:26])
	h.EncodeMs = binary.BigEndian.Uint16(buf[26:28])
	h.JPEGLength = binary.BigEndian.Uint32(buf[28:32])
	if h.JPEGLength == 0 || h.JPEGLength > 16*1024*1024 {
		return h, fmt.Errorf("invalid jpeg length %d", h.JPEGLength)
	}
	return h, nil
}

// WriteFrame writes one framed JPEG to w.
func WriteFrame(w io.Writer, f Frame) error {
	f.Header.JPEGLength = uint32(len(f.JPEG))
	if f.Header.Version == 0 {
		f.Header.Version = Version
	}
	var hdr [HeaderSize]byte
	EncodeHeader(f.Header, hdr[:])
	if _, err := w.Write(hdr[:]); err != nil {
		return err
	}
	_, err := w.Write(f.JPEG)
	return err
}

// ReadFrame reads one complete frame using io.ReadFull. A short read is an error.
func ReadFrame(r io.Reader) (*Frame, error) {
	start := time.Now()
	var hdr [HeaderSize]byte
	if _, err := io.ReadFull(r, hdr[:]); err != nil {
		return nil, err
	}
	h, err := DecodeHeader(hdr[:])
	if err != nil {
		return nil, err
	}
	jpeg := make([]byte, h.JPEGLength)
	if _, err := io.ReadFull(r, jpeg); err != nil {
		return nil, fmt.Errorf("short jpeg payload: %w", err)
	}
	return &Frame{
		Header:     h,
		JPEG:       jpeg,
		Hash:       HashJPEG(jpeg),
		ReceivedAt: time.Now(),
		TransferMs: time.Since(start).Milliseconds(),
	}, nil
}

// HashJPEG returns an 8-byte hex fingerprint of the JPEG bytes.
func HashJPEG(jpeg []byte) string {
	sum := sha1.Sum(jpeg)
	return hex.EncodeToString(sum[:8])
}
