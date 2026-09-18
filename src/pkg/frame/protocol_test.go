package frame

import (
	"bytes"
	"io"
	"testing"
)

func sampleFrame(seq uint32, jpeg []byte) Frame {
	return Frame{
		Header: Header{
			Version:      Version,
			Rotation:     1,
			Seq:          seq,
			CapturedAtMs: 1_700_000_000_000,
			DeviceWidth:  1080,
			DeviceHeight: 2340,
			ImageWidth:   720,
			ImageHeight:  1560,
			EncodeMs:     12,
		},
		JPEG: jpeg,
	}
}

func TestWriteReadRoundTrip(t *testing.T) {
	jpeg := []byte{0xFF, 0xD8, 0xFF, 0x01, 0x02, 0x03}
	var buf bytes.Buffer
	if err := WriteFrame(&buf, sampleFrame(7, jpeg)); err != nil {
		t.Fatal(err)
	}
	got, err := ReadFrame(&buf)
	if err != nil {
		t.Fatal(err)
	}
	if got.Seq != 7 || got.Rotation != 1 || got.DeviceWidth != 1080 || got.ImageWidth != 720 {
		t.Fatalf("header %+v", got.Header)
	}
	if !bytes.Equal(got.JPEG, jpeg) {
		t.Fatalf("jpeg %v", got.JPEG)
	}
	if got.Hash != HashJPEG(jpeg) {
		t.Fatalf("hash %s", got.Hash)
	}
}

func TestReadFrameShortHeader(t *testing.T) {
	_, err := ReadFrame(bytes.NewReader([]byte("ADB")))
	if err == nil {
		t.Fatal("expected short header error")
	}
}

func TestReadFrameShortJPEG(t *testing.T) {
	var buf bytes.Buffer
	if err := WriteFrame(&buf, sampleFrame(1, []byte{1, 2, 3, 4, 5})); err != nil {
		t.Fatal(err)
	}
	partial := buf.Bytes()[:HeaderSize+2]
	_, err := ReadFrame(bytes.NewReader(partial))
	if err == nil {
		t.Fatal("expected short jpeg error")
	}
}

func TestReadFrameDisconnect(t *testing.T) {
	_, err := ReadFrame(bytes.NewReader(nil))
	if err != io.EOF {
		t.Fatalf("got %v, want EOF", err)
	}
}

func TestDecodeHeaderRejectsBadMagic(t *testing.T) {
	buf := make([]byte, HeaderSize)
	EncodeHeader(Header{Version: Version, JPEGLength: 4}, buf)
	buf[0] = 'X'
	if _, err := DecodeHeader(buf); err == nil {
		t.Fatal("expected bad magic")
	}
}
