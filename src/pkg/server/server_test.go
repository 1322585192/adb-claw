package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"strings"
	"testing"

	"github.com/llm-net/adb-claw/pkg/adb"
)

const sampleXML = `<?xml version="1.0" encoding="UTF-8"?>
<hierarchy rotation="0">
  <node index="0" text="Login" resource-id="com.example:id/btn_login" class="android.widget.Button" package="com.example" content-desc="" checkable="false" checked="false" clickable="true" enabled="true" focusable="true" focused="false" scrollable="false" selected="false" bounds="[200,500][880,600]"></node>
</hierarchy>`

type mockCmd struct {
	png    []byte
	taps   []string
	shells [][]string
	shellN int
	execN  int
}

func (m *mockCmd) calls() [][]string { return m.shells }

func (m *mockCmd) Shell(args ...string) (*adb.Result, error) {
	m.shellN++
	m.shells = append(m.shells, append([]string{}, args...))
	if len(args) == 1 && strings.Contains(args[0], "uiautomator dump") {
		return &adb.Result{Stdout: sampleXML}, nil
	}
	if len(args) >= 2 && args[0] == "sh" && args[1] == "-c" {
		return &adb.Result{Stdout: sampleXML}, nil
	}
	if len(args) >= 1 && args[0] == "input" {
		m.taps = append(m.taps, strings.Join(args, " "))
		return &adb.Result{}, nil
	}
	return &adb.Result{}, nil
}

func (m *mockCmd) ExecOut(args ...string) ([]byte, error) {
	m.execN++
	if len(args) >= 1 && args[0] == "screencap" {
		return m.png, nil
	}
	return nil, fmt.Errorf("unexpected exec-out %v", args)
}

func (m *mockCmd) RawCommand(args ...string) (*adb.Result, error) {
	return &adb.Result{}, nil
}

func solidPNG(w, h int) []byte {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.RGBA{R: 10, G: 20, B: 30, A: 255})
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		panic(err)
	}
	return buf.Bytes()
}

func TestServerObserveThenActUsesCache(t *testing.T) {
	t.Setenv("TMPDIR", t.TempDir())
	cmd := &mockCmd{png: solidPNG(40, 60)}
	in := bytes.NewBufferString(`{"id":1,"method":"observe","params":{"skip_screenshot":true}}
{"id":2,"method":"act","params":{"action":"tap","index":0}}
`)
	var out bytes.Buffer
	srv := &Server{Cmd: cmd, Serial: "test", In: in, Out: &out}
	if err := srv.Run(); err != nil {
		t.Fatal(err)
	}

	dec := json.NewDecoder(&out)
	var observeResp Response
	for {
		if err := dec.Decode(&observeResp); err != nil {
			t.Fatal(err)
		}
		if len(observeResp.ID) > 0 {
			break
		}
	}
	if observeResp.Error != nil {
		t.Fatalf("observe error: %+v", observeResp.Error)
	}
	raw, _ := json.Marshal(observeResp.Result)
	if !strings.Contains(string(raw), `"state_id"`) {
		t.Fatalf("observe missing state_id: %s", raw)
	}

	var actResp Response
	if err := dec.Decode(&actResp); err != nil {
		t.Fatal(err)
	}
	if actResp.Error != nil {
		t.Fatalf("act error: %+v", actResp.Error)
	}
	dumps := 0
	for _, c := range cmd.calls() {
		if len(c) == 1 && strings.Contains(c[0], "uiautomator dump") {
			dumps++
		}
		if len(c) >= 2 && c[0] == "sh" {
			dumps++
		}
	}
	if dumps != 1 {
		t.Fatalf("expected 1 UI dump, got %d (shells=%d taps=%v)", dumps, cmd.shellN, cmd.taps)
	}
	if len(cmd.taps) != 1 || !strings.Contains(cmd.taps[0], "tap 540 550") {
		t.Fatalf("tap args = %v", cmd.taps)
	}
}

func TestServerActWithoutObserveIsStale(t *testing.T) {
	t.Setenv("TMPDIR", t.TempDir())
	cmd := &mockCmd{png: solidPNG(10, 10)}
	in := bytes.NewBufferString(`{"id":1,"method":"act","params":{"action":"tap","index":0}}
`)
	var out bytes.Buffer
	srv := &Server{Cmd: cmd, Serial: "none", In: in, Out: &out}
	if err := srv.Run(); err != nil {
		t.Fatal(err)
	}
	var resp Response
	if err := json.NewDecoder(&out).Decode(&resp); err != nil {
		t.Fatal(err)
	}
	if resp.Error == nil || resp.Error.Code != "STALE_STATE" {
		t.Fatalf("expected STALE_STATE, got %+v", resp.Error)
	}
	if len(cmd.taps) != 0 {
		t.Fatalf("should not tap on stale state: %v", cmd.taps)
	}
}

func TestSessionRejectsWrongStateID(t *testing.T) {
	var s Session
	_, err := s.Get("abc", 0)
	if err == nil {
		t.Fatal("expected empty session error")
	}
}
