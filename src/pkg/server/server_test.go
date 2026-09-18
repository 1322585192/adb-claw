package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"strings"
	"testing"
	"time"

	"github.com/llm-net/adb-claw/pkg/adb"
	"github.com/llm-net/adb-claw/pkg/coord"
	"github.com/llm-net/adb-claw/pkg/frame"
)

type mockCmd struct {
	taps   []string
	shells [][]string
}

func (m *mockCmd) Shell(args ...string) (*adb.Result, error) {
	m.shells = append(m.shells, append([]string{}, args...))
	if strings.Contains(strings.Join(args, " "), "ADBClawInput") {
		return &adb.Result{Stdout: "OK\n"}, nil
	}
	if len(args) >= 1 && args[0] == "input" {
		m.taps = append(m.taps, strings.Join(args, " "))
		return &adb.Result{}, nil
	}
	if len(args) >= 1 && args[0] == "wm" {
		return &adb.Result{Stdout: "Physical size: 1080x2340\n"}, nil
	}
	return &adb.Result{}, nil
}
func (m *mockCmd) ExecOut(args ...string) ([]byte, error) {
	return nil, fmt.Errorf("unused")
}
func (m *mockCmd) RawCommand(args ...string) (*adb.Result, error) {
	return &adb.Result{}, nil
}

func jpegFrame(seq uint32, w, h int, seed byte) *frame.Frame {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.RGBA{R: seed, A: 255})
		}
	}
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 40}); err != nil {
		panic(err)
	}
	jpegBytes := buf.Bytes()
	now := time.Now()
	return &frame.Frame{
		Header: frame.Header{
			Version:      frame.Version,
			Seq:          seq,
			CapturedAtMs: uint64(now.UnixMilli()),
			DeviceWidth:  1080,
			DeviceHeight: 2340,
			ImageWidth:   uint16(w),
			ImageHeight:  uint16(h),
			JPEGLength:   uint32(len(jpegBytes)),
		},
		JPEG:       jpegBytes,
		Hash:       frame.HashJPEG(jpegBytes),
		ReceivedAt: now,
		Source:     "test",
	}
}

func TestFrameLatestThenNormalizedAct(t *testing.T) {
	t.Setenv("TMPDIR", t.TempDir())
	cmd := &mockCmd{}
	f := jpegFrame(4, 40, 80, 10)
	srv := &Server{Cmd: cmd, Serial: "test", Options: Options{LatestPath: t.TempDir() + "/latest.jpg", MaxAge: time.Minute}}
	srv.session.Put(f)

	in := bytes.NewBufferString(`{"id":1,"method":"frame.latest"}
{"id":2,"method":"act","params":{"action":"tap","frame_seq":4,"x":500,"y":500}}
`)
	var out bytes.Buffer
	srv.In, srv.Out = in, &out
	if err := srv.Run(); err != nil {
		t.Fatal(err)
	}

	dec := json.NewDecoder(&out)
	var latest Response
	if err := dec.Decode(&latest); err != nil {
		t.Fatal(err)
	}
	if latest.Error != nil {
		t.Fatalf("latest: %+v", latest.Error)
	}
	raw, _ := json.Marshal(latest.Result)
	if !strings.Contains(string(raw), `"frame_seq"`) || strings.Contains(string(raw), "base64") {
		t.Fatalf("latest result %s", raw)
	}

	var act Response
	if err := dec.Decode(&act); err != nil {
		t.Fatal(err)
	}
	if act.Error != nil {
		t.Fatalf("act: %+v", act.Error)
	}
	want := coord.Denormalize(500, 500, 1080, 2340)
	if len(cmd.taps) != 1 || !strings.Contains(cmd.taps[0], fmt.Sprintf("tap %d %d", want.X, want.Y)) {
		t.Fatalf("normalized 500,500 on 1080x2340 should tap (%d,%d), got %v", want.X, want.Y, cmd.taps)
	}
	for _, sh := range cmd.shells {
		joined := strings.Join(sh, " ")
		if strings.Contains(joined, "uiautomator") {
			t.Fatalf("act must not dump UI: %v", cmd.shells)
		}
	}
}

func TestActStaleUnknownSeq(t *testing.T) {
	t.Setenv("TMPDIR", t.TempDir())
	cmd := &mockCmd{}
	srv := &Server{Cmd: cmd, Options: Options{LatestPath: t.TempDir() + "/l.jpg"}}
	in := bytes.NewBufferString(`{"id":1,"method":"act","params":{"action":"tap","frame_seq":9,"x":1,"y":1}}
`)
	var out bytes.Buffer
	srv.In, srv.Out = in, &out
	if err := srv.Run(); err != nil {
		t.Fatal(err)
	}
	var resp Response
	if err := json.NewDecoder(&out).Decode(&resp); err != nil {
		t.Fatal(err)
	}
	if resp.Error == nil || resp.Error.Code != "STALE_FRAME" {
		t.Fatalf("expected STALE_FRAME, got %+v", resp.Error)
	}
	if len(cmd.taps) != 0 {
		t.Fatalf("should not tap: %v", cmd.taps)
	}
}

func TestActStaleWhenHashChanged(t *testing.T) {
	cmd := &mockCmd{}
	srv := &Server{Cmd: cmd, Options: Options{MaxAge: time.Minute}}
	old := jpegFrame(1, 20, 20, 1)
	srv.session.Put(old)
	newer := jpegFrame(2, 20, 20, 90)
	srv.session.Put(newer)

	in := bytes.NewBufferString(`{"id":1,"method":"act","params":{"action":"tap","frame_seq":1,"x":10,"y":10}}
`)
	var out bytes.Buffer
	srv.In, srv.Out = in, &out
	if err := srv.Run(); err != nil {
		t.Fatal(err)
	}
	var resp Response
	if err := json.NewDecoder(&out).Decode(&resp); err != nil {
		t.Fatal(err)
	}
	if resp.Error == nil || resp.Error.Code != "STALE_FRAME" {
		t.Fatalf("expected STALE_FRAME on visual change, got %+v", resp.Error)
	}
}

func TestSessionResolveSameHashAllowsNewerSeqGap(t *testing.T) {
	var s Session
	f1 := jpegFrame(1, 10, 10, 7)
	s.Put(f1)
	f2 := *f1
	f2.Seq = 2
	f2.Header.Seq = 2
	s.Put(&f2)
	if _, err := s.Resolve(1, time.Minute); err != nil {
		t.Fatalf("same visual should be actionable: %v", err)
	}
}

func TestFrameWaitAfterReturnsReadablePathWithoutLatest(t *testing.T) {
	t.Setenv("TMPDIR", t.TempDir())
	f := jpegFrame(4, 40, 80, 10)
	srv := &Server{
		Cmd:     &mockCmd{},
		Options: Options{LatestPath: t.TempDir() + "/latest.jpg"},
	}
	srv.session.Put(f)
	srv.In = bytes.NewBufferString(
		`{"id":1,"method":"frame.wait_after","params":{"frame_seq":4,"timeout_ms":1}}` + "\n",
	)
	var out bytes.Buffer
	srv.Out = &out
	if err := srv.Run(); err != nil {
		t.Fatal(err)
	}

	var response Response
	if err := json.NewDecoder(&out).Decode(&response); err != nil {
		t.Fatal(err)
	}
	if response.Error != nil {
		t.Fatalf("wait_after: %+v", response.Error)
	}
	result, ok := response.Result.(map[string]interface{})
	if !ok {
		t.Fatalf("result type = %T", response.Result)
	}
	if result["path"] == "" || result["next"] != "read_path_directly" {
		t.Fatalf("wait_after result must be directly consumable: %#v", result)
	}
	if changed, ok := result["changed"].(bool); !ok || changed {
		t.Fatalf("same frame should report changed=false: %#v", result)
	}
}

func TestActTypesUnicodeWithBuiltInHelper(t *testing.T) {
	cmd := &mockCmd{}
	srv := &Server{Cmd: cmd}
	srv.In = bytes.NewBufferString(
		`{"id":1,"method":"act","params":{"action":"type","text":"王者荣耀"}}` + "\n",
	)
	var out bytes.Buffer
	srv.Out = &out
	if err := srv.Run(); err != nil {
		t.Fatal(err)
	}

	var response Response
	if err := json.NewDecoder(&out).Decode(&response); err != nil {
		t.Fatal(err)
	}
	if response.Error != nil {
		t.Fatalf("Unicode type: %+v", response.Error)
	}
	raw, _ := json.Marshal(response.Result)
	if !strings.Contains(string(raw), `"method":"unicode_clipboard"`) {
		t.Fatalf("response must expose Unicode transport: %s", raw)
	}
	joined := fmt.Sprint(cmd.shells)
	if !strings.Contains(joined, "ADBClawInput") ||
		!strings.Contains(joined, "KEYCODE_PASTE") {
		t.Fatalf("missing helper/paste calls: %v", cmd.shells)
	}
}
