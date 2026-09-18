package server

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	"github.com/llm-net/adb-claw/pkg/adb"
	"github.com/llm-net/adb-claw/pkg/input"
	"github.com/llm-net/adb-claw/pkg/observe"
)

// Options configure a persistent stdio session.
type Options struct {
	CaptureMode observe.CaptureMode
	UIMode      observe.UIMode
	Compressed  bool
	MaxWidth    int
	Quality     int
	MaxAge      time.Duration
}

// Server reads JSONL requests and writes JSONL responses/events.
type Server struct {
	Cmd     adb.Commander
	Serial  string
	In      io.Reader
	Out     io.Writer
	Options Options
	session Session
}

// Run processes requests until stdin closes.
func (s *Server) Run() error {
	if s.Options.MaxAge == 0 {
		s.Options.MaxAge = observe.DefaultSnapshotMaxAge
	}
	if s.Options.Quality == 0 {
		s.Options.Quality = observe.DefaultJPEGQuality
	}
	dec := json.NewDecoder(bufio.NewReader(s.In))
	enc := json.NewEncoder(s.Out)
	for {
		var req Request
		if err := dec.Decode(&req); err != nil {
			if err == io.EOF {
				return nil
			}
			if err := enc.Encode(rpcError(nil, "INVALID_REQUEST", err.Error())); err != nil {
				return err
			}
			continue
		}
		for _, resp := range s.handle(req) {
			if err := enc.Encode(resp); err != nil {
				return err
			}
		}
	}
}

type observeParams struct {
	Width      int    `json:"width"`
	Quality    int    `json:"quality"`
	Format     string `json:"format"`
	Path       string `json:"path"`
	Capture    string `json:"capture"`
	UIMode     string `json:"ui_mode"`
	Compressed bool   `json:"compressed"`
	SkipImage  bool   `json:"skip_screenshot"`
	Profile    bool   `json:"profile"`
}

type actParams struct {
	StateID    string `json:"state_id"`
	Action     string `json:"action"`
	Index      *int   `json:"index"`
	Handle     string `json:"handle"`
	ID         string `json:"id"`
	Text       string `json:"text"`
	X          *int   `json:"x"`
	Y          *int   `json:"y"`
	X2         *int   `json:"x2"`
	Y2         *int   `json:"y2"`
	DurationMs int    `json:"duration_ms"`
	Key        string `json:"key"`
}

func (s *Server) handle(req Request) []Response {
	switch strings.ToLower(req.Method) {
	case "ping":
		return []Response{rpcResult(req.ID, map[string]string{"ok": "pong"})}
	case "observe":
		return s.handleObserve(req)
	case "act":
		return []Response{s.handleAct(req)}
	case "close":
		return []Response{rpcResult(req.ID, map[string]string{"ok": "closed"})}
	default:
		return []Response{rpcError(req.ID, "UNKNOWN_METHOD", "unknown method "+req.Method)}
	}
}

func (s *Server) handleObserve(req Request) []Response {
	params, err := decodeParams[observeParams](req.Params)
	if err != nil {
		return []Response{rpcError(req.ID, "INVALID_ARGS", err.Error())}
	}
	opts := observe.ObserveOptions{
		MaxWidth:   params.Width,
		Format:     params.Format,
		Quality:    params.Quality,
		Path:       params.Path,
		Mode:       observe.CaptureMode(params.Capture),
		UIMode:     observe.UIMode(params.UIMode),
		Compressed: params.Compressed,
		Profile:    params.Profile,
		SkipImage:  params.SkipImage,
	}
	if opts.MaxWidth == 0 {
		opts.MaxWidth = s.Options.MaxWidth
	}
	if opts.Quality == 0 {
		opts.Quality = s.Options.Quality
	}
	if opts.Mode == "" {
		opts.Mode = s.Options.CaptureMode
	}
	if opts.UIMode == "" {
		opts.UIMode = s.Options.UIMode
	}

	var (
		events []Response
		evMu   sync.Mutex
	)
	result := observe.ObserveStream(s.Cmd, opts, func(ev observe.ObserveEvent) {
		evMu.Lock()
		defer evMu.Unlock()
		if ev.Err != nil {
			events = append(events, rpcEvent("event.error", map[string]string{"kind": ev.Kind, "error": ev.Err.Error()}))
			return
		}
		if ev.Kind == "screenshot" {
			events = append(events, rpcEvent("event.screenshot", ev.Screenshot))
			return
		}
		events = append(events, rpcEvent("event.ui", ev.UI))
	})
	if result.UI != nil {
		s.session.Put(&observe.Snapshot{
			StateID:    result.UI.StateID,
			Serial:     s.Serial,
			CapturedAt: time.Now().UTC(),
			Tree:       result.UI,
		})
	}
	if result.Screenshot == nil && result.UI == nil {
		msg := "observe failed"
		if len(result.Errors) > 0 {
			msg = strings.Join(result.Errors, "; ")
		}
		return append(events, rpcError(req.ID, "OBSERVE_FAILED", msg))
	}
	return append(events, rpcResult(req.ID, result))
}

func (s *Server) handleAct(req Request) Response {
	params, err := decodeParams[actParams](req.Params)
	if err != nil {
		return rpcError(req.ID, "INVALID_ARGS", err.Error())
	}
	action := strings.ToLower(params.Action)
	if action == "" {
		action = "tap"
	}

	if action == "key" {
		if params.Key == "" {
			return rpcError(req.ID, "INVALID_ARGS", "key action requires key")
		}
		if err := input.KeyEvent(s.Cmd, params.Key); err != nil {
			return rpcError(req.ID, "KEY_FAILED", err.Error())
		}
		return rpcResult(req.ID, map[string]interface{}{"action": "key", "key": params.Key})
	}

	x, y, info, err := s.resolveTarget(params)
	if err != nil {
		code := "ELEMENT_NOT_FOUND"
		if _, ok := err.(staleError); ok || strings.Contains(err.Error(), "snapshot") || strings.Contains(err.Error(), "stale") || strings.Contains(err.Error(), "expired") {
			code = "STALE_STATE"
		}
		return rpcError(req.ID, code, err.Error())
	}

	switch action {
	case "tap":
		if err := input.Tap(s.Cmd, x, y); err != nil {
			return rpcError(req.ID, "TAP_FAILED", err.Error())
		}
	case "long-press", "long_press":
		dur := params.DurationMs
		if dur <= 0 {
			dur = 1000
		}
		if err := input.LongPress(s.Cmd, x, y, dur); err != nil {
			return rpcError(req.ID, "LONG_PRESS_FAILED", err.Error())
		}
	case "swipe":
		if params.X2 == nil || params.Y2 == nil {
			return rpcError(req.ID, "INVALID_ARGS", "swipe requires x2/y2")
		}
		dur := params.DurationMs
		if dur <= 0 {
			dur = 300
		}
		if err := input.Swipe(s.Cmd, x, y, *params.X2, *params.Y2, dur); err != nil {
			return rpcError(req.ID, "SWIPE_FAILED", err.Error())
		}
	default:
		return rpcError(req.ID, "UNKNOWN_ACTION", "unsupported action "+params.Action)
	}

	data := map[string]interface{}{"action": action, "x": x, "y": y}
	if info != nil {
		data["element"] = info
	}
	return rpcResult(req.ID, data)
}

func (s *Server) resolveTarget(params actParams) (int, int, map[string]interface{}, error) {
	if params.X != nil && params.Y != nil {
		return *params.X, *params.Y, nil, nil
	}

	snap, err := s.session.Get(params.StateID, s.Options.MaxAge)
	if err != nil {
		// Fall back to on-disk snapshot from CLI observe.
		disk, diskErr := observe.LoadSnapshotByID(s.Serial, params.StateID, s.Options.MaxAge)
		if diskErr != nil {
			if err != nil {
				return 0, 0, nil, err
			}
			return 0, 0, nil, diskErr
		}
		snap = disk
		s.session.Put(disk)
	}

	var el *observe.Element
	switch {
	case params.Handle != "":
		el, err = snap.Tree.FindByHandle(params.Handle)
	case params.Index != nil:
		el, err = snap.Tree.FindByIndex(*params.Index)
	case params.ID != "":
		found := snap.Tree.FindByID(params.ID)
		if len(found) == 0 {
			err = fmt.Errorf("no element found with id '%s'", params.ID)
		} else {
			el = &found[0]
		}
	case params.Text != "":
		found := snap.Tree.FindByText(params.Text)
		if len(found) == 0 {
			err = fmt.Errorf("no element found with text '%s'", params.Text)
		} else {
			el = &found[0]
		}
	default:
		return 0, 0, nil, fmt.Errorf("act requires coordinates or an element selector")
	}
	if err != nil {
		return 0, 0, nil, err
	}
	return el.Center.X, el.Center.Y, map[string]interface{}{
		"index":  el.Index,
		"handle": el.Handle,
		"text":   el.Text,
	}, nil
}
