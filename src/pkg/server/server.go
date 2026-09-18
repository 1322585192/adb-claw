package server

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/llm-net/adb-claw/pkg/adb"
	"github.com/llm-net/adb-claw/pkg/coord"
	"github.com/llm-net/adb-claw/pkg/device"
	"github.com/llm-net/adb-claw/pkg/frame"
	"github.com/llm-net/adb-claw/pkg/input"
	"github.com/llm-net/adb-claw/pkg/observe"
)

// DefaultMaxFrameAge is how long a model-visible frame remains actionable.
const DefaultMaxFrameAge = 3 * time.Second

// Options configure a persistent stdio session.
type Options struct {
	Width      int
	Quality    int
	MaxAge     time.Duration
	ForceHigh  bool
	LatestPath string
}

// Server reads JSONL requests and writes JSONL responses.
type Server struct {
	Cmd     adb.Commander
	Client  *adb.Client
	Serial  string
	In      io.Reader
	Out     io.Writer
	Options Options
	Source  *frame.Source
	session Session
}

// Run processes requests until stdin closes.
func (s *Server) Run() error {
	if s.Options.MaxAge == 0 {
		s.Options.MaxAge = DefaultMaxFrameAge
	}
	if s.Options.Quality == 0 {
		s.Options.Quality = frame.QualityHigh
	}
	if s.Options.Width == 0 {
		s.Options.Width = frame.WidthHigh
	}
	if s.Options.LatestPath == "" {
		s.Options.LatestPath = frame.DefaultLatestPath()
	}
	if s.Source == nil && s.Client != nil {
		src, err := frame.Start(context.Background(), s.Client, frame.Options{
			Width:      s.Options.Width,
			Quality:    s.Options.Quality,
			LatestPath: s.Options.LatestPath,
			ForceHigh:  s.Options.ForceHigh,
		})
		if err != nil {
			return err
		}
		s.Source = src
		defer s.Source.Close()
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
		resp := s.handle(req)
		if err := enc.Encode(resp); err != nil {
			return err
		}
		if strings.ToLower(req.Method) == "close" {
			return nil
		}
	}
}

type frameParams struct {
	Seq       uint32 `json:"frame_seq"`
	TimeoutMs int    `json:"timeout_ms"`
}

type actParams struct {
	FrameSeq   uint32 `json:"frame_seq"`
	Action     string `json:"action"`
	X          *int   `json:"x"`
	Y          *int   `json:"y"`
	X2         *int   `json:"x2"`
	Y2         *int   `json:"y2"`
	DurationMs int    `json:"duration_ms"`
	Key        string `json:"key"`
	Text       string `json:"text"`
	Direction  string `json:"direction"`
	Normalized *bool  `json:"normalized"`
}

func (s *Server) handle(req Request) Response {
	switch strings.ToLower(req.Method) {
	case "ping":
		return rpcResult(req.ID, map[string]string{"ok": "pong"})
	case "frame.latest":
		return s.handleLatest(req)
	case "frame.wait_after":
		return s.handleWaitAfter(req)
	case "act":
		return s.handleAct(req)
	case "device.info":
		return s.handleDeviceInfo(req)
	case "close":
		return rpcResult(req.ID, map[string]string{"ok": "closed"})
	default:
		return rpcError(req.ID, "UNKNOWN_METHOD", "unknown method "+req.Method)
	}
}

func (s *Server) handleLatest(req Request) Response {
	f, err := s.latestFrame(800 * time.Millisecond)
	if err != nil {
		return rpcError(req.ID, "FRAME_UNAVAILABLE", err.Error())
	}
	s.session.Put(f)
	info, err := s.writeLatest(f)
	if err != nil {
		return rpcError(req.ID, "FRAME_WRITE_FAILED", err.Error())
	}
	return rpcResult(req.ID, info)
}

func (s *Server) handleWaitAfter(req Request) Response {
	params, err := decodeParams[frameParams](req.Params)
	if err != nil {
		return rpcError(req.ID, "INVALID_ARGS", err.Error())
	}
	timeout := time.Duration(params.TimeoutMs) * time.Millisecond
	if timeout <= 0 {
		timeout = 2 * time.Second
	}
	hash := ""
	if meta := s.sessionLookup(params.Seq); meta != nil {
		hash = meta.Hash
	}
	var f *frame.Frame
	if s.Source != nil {
		f = s.Source.WaitAfter(params.Seq, hash, timeout)
	} else {
		f = s.session.Latest()
	}
	if f == nil {
		return rpcError(req.ID, "FRAME_UNAVAILABLE", "no frame")
	}
	s.session.Put(f)
	info, err := s.writeLatest(f)
	if err != nil {
		return rpcError(req.ID, "FRAME_WRITE_FAILED", err.Error())
	}
	info["changed"] = hash == "" || f.Hash != hash
	info["next"] = "read_path_directly"
	return rpcResult(req.ID, info)
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
		if err := input.KeyEvent(s.commander(), params.Key); err != nil {
			return rpcError(req.ID, "KEY_FAILED", err.Error())
		}
		return rpcResult(req.ID, map[string]interface{}{"action": "key", "key": params.Key})
	}
	if action == "type" {
		if params.Text == "" {
			return rpcError(req.ID, "INVALID_ARGS", "type action requires text")
		}
		method, err := input.TypeText(s.commander(), params.Text)
		if err != nil {
			return rpcError(req.ID, "TYPE_FAILED", err.Error())
		}
		return rpcResult(req.ID, map[string]interface{}{
			"action": "type",
			"text":   params.Text,
			"method": method,
		})
	}

	meta, err := s.session.Resolve(params.FrameSeq, s.Options.MaxAge)
	if err != nil {
		return rpcError(req.ID, "STALE_FRAME", err.Error())
	}

	normalized := true
	if params.Normalized != nil {
		normalized = *params.Normalized
	}

	x, y, err := s.mapPoint(params.X, params.Y, meta, normalized, "x", "y")
	if err != nil {
		return rpcError(req.ID, "INVALID_ARGS", err.Error())
	}

	switch action {
	case "tap", "click":
		if err := input.Tap(s.commander(), x, y); err != nil {
			return rpcError(req.ID, "TAP_FAILED", err.Error())
		}
	case "long-press", "long_press":
		dur := params.DurationMs
		if dur <= 0 {
			dur = 1000
		}
		if err := input.LongPress(s.commander(), x, y, dur); err != nil {
			return rpcError(req.ID, "LONG_PRESS_FAILED", err.Error())
		}
	case "swipe", "scroll":
		x2, y2, err := s.mapPoint(params.X2, params.Y2, meta, normalized, "x2", "y2")
		if err != nil {
			if action == "scroll" && params.Direction != "" {
				w, h := meta.DeviceWidth, meta.DeviceHeight
				dist := h * 60 / 100
				if params.Direction == "left" || params.Direction == "right" {
					dist = w * 60 / 100
				}
				var e error
				x, y, x2, y2, e = input.ScrollDirection(w, h, dist, params.Direction)
				if e != nil {
					return rpcError(req.ID, "INVALID_ARGS", e.Error())
				}
			} else {
				return rpcError(req.ID, "INVALID_ARGS", err.Error())
			}
		}
		dur := params.DurationMs
		if dur <= 0 {
			dur = 300
		}
		if err := input.Swipe(s.commander(), x, y, x2, y2, dur); err != nil {
			return rpcError(req.ID, "SWIPE_FAILED", err.Error())
		}
		return rpcResult(req.ID, map[string]interface{}{
			"action":    action,
			"x":         x,
			"y":         y,
			"x2":        x2,
			"y2":        y2,
			"frame_seq": params.FrameSeq,
		})
	default:
		return rpcError(req.ID, "UNKNOWN_ACTION", "unsupported action "+params.Action)
	}

	return rpcResult(req.ID, map[string]interface{}{
		"action":    action,
		"x":         x,
		"y":         y,
		"frame_seq": params.FrameSeq,
	})
}

func (s *Server) handleDeviceInfo(req Request) Response {
	info := map[string]interface{}{
		"serial": s.Serial,
	}
	if cmd := s.commander(); cmd != nil {
		if w, h, err := input.CurrentScreenSize(cmd); err == nil {
			info["width"] = w
			info["height"] = h
		}
		if st, err := device.GetScreenStatus(cmd); err == nil {
			info["rotation"] = st.Rotation
			info["display"] = st.Display
		}
	}
	if s.Source != nil {
		info["frame_source"] = s.Source.Mode()
		info["resolution"] = s.Source.Resolution()
	}
	return rpcResult(req.ID, info)
}

func (s *Server) mapPoint(x, y *int, meta *frame.FrameMeta, normalized bool, xn, yn string) (int, int, error) {
	if x == nil || y == nil {
		return 0, 0, fmt.Errorf("%s/%s required", xn, yn)
	}
	if !normalized {
		return *x, *y, nil
	}
	if err := coord.ValidateNormalized(xn, *x); err != nil {
		return 0, 0, err
	}
	if err := coord.ValidateNormalized(yn, *y); err != nil {
		return 0, 0, err
	}
	p := coord.Denormalize(*x, *y, meta.DeviceWidth, meta.DeviceHeight)
	return p.X, p.Y, nil
}

func (s *Server) latestFrame(wait time.Duration) (*frame.Frame, error) {
	if s.Source != nil {
		deadline := time.Now().Add(wait)
		for time.Now().Before(deadline) {
			if f := s.Source.Latest(); f != nil {
				return f, nil
			}
			time.Sleep(20 * time.Millisecond)
		}
		if f := s.Source.Latest(); f != nil {
			return f, nil
		}
	}
	if f := s.session.Latest(); f != nil {
		return f, nil
	}
	if s.Cmd != nil {
		f, err := captureOne(s.Cmd, 1, s.Options.Width, s.Options.Quality)
		if err != nil {
			return nil, err
		}
		return f, nil
	}
	return nil, fmt.Errorf("no frame source")
}

func (s *Server) writeLatest(f *frame.Frame) (map[string]interface{}, error) {
	path := s.Options.LatestPath
	if err := frame.WriteAtomic(path, f.JPEG); err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"frame_seq":     f.Seq,
		"captured_at":   f.CapturedAt().Format(time.RFC3339Nano),
		"frame_age_ms":  f.Age().Milliseconds(),
		"rotation":      f.Rotation,
		"device_width":  f.DeviceWidth,
		"device_height": f.DeviceHeight,
		"image_width":   f.ImageWidth,
		"image_height":  f.ImageHeight,
		"scale":         f.Scale(),
		"path":          path,
		"size_bytes":    len(f.JPEG),
		"hash":          f.Hash,
		"resolution":    f.ResolutionBucket(),
		"source":        f.Source,
	}, nil
}

func (s *Server) sessionLookup(seq uint32) *frame.FrameMeta {
	s.session.mu.Lock()
	defer s.session.mu.Unlock()
	for i := range s.session.history {
		if s.session.history[i].Seq == seq {
			m := s.session.history[i]
			return &m
		}
	}
	return nil
}

func (s *Server) commander() adb.Commander {
	if s.Cmd != nil {
		return s.Cmd
	}
	if s.Client != nil {
		return s.Client
	}
	return nil
}

func captureOne(cmd adb.Commander, seq uint32, width, quality int) (*frame.Frame, error) {
	res, err := observe.CaptureScreenshot(cmd, observe.CaptureOptions{
		MaxWidth: width,
		Format:   "jpeg",
		Quality:  quality,
		Mode:     observe.CaptureModeAuto,
	})
	if err != nil {
		return nil, err
	}
	now := time.Now()
	return &frame.Frame{
		Header: frame.Header{
			Version:      frame.Version,
			Seq:          seq,
			CapturedAtMs: uint64(now.UnixMilli()),
			DeviceWidth:  uint16(res.DeviceWidth),
			DeviceHeight: uint16(res.DeviceHeight),
			ImageWidth:   uint16(res.ImageWidth),
			ImageHeight:  uint16(res.ImageHeight),
			JPEGLength:   uint32(len(res.Bytes)),
		},
		JPEG:       res.Bytes,
		Hash:       frame.HashJPEG(res.Bytes),
		ReceivedAt: now,
		Source:     res.Mode,
	}, nil
}
