package server

import (
	"sync"
	"time"

	"github.com/llm-net/adb-claw/pkg/frame"
)

// Session remembers recent frames so act(frame_seq) can detect staleness.
type Session struct {
	mu      sync.Mutex
	latest  *frame.Frame
	history []frame.FrameMeta
}

func (s *Session) Put(f *frame.Frame) {
	if f == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.latest = f
	s.history = append(s.history, frame.FrameMeta{
		Seq:          f.Seq,
		Hash:         f.Hash,
		CapturedAt:   f.CapturedAt(),
		DeviceWidth:  int(f.DeviceWidth),
		DeviceHeight: int(f.DeviceHeight),
		Rotation:     int(f.Rotation),
	})
	if len(s.history) > 32 {
		s.history = s.history[len(s.history)-32:]
	}
}

func (s *Session) Latest() *frame.Frame {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.latest
}

func (s *Session) Resolve(seq uint32, maxAge time.Duration) (*frame.FrameMeta, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.latest == nil {
		return nil, errStale("no frame yet; call frame.latest first")
	}
	var meta *frame.FrameMeta
	for i := range s.history {
		if s.history[i].Seq == seq {
			m := s.history[i]
			meta = &m
			break
		}
	}
	if meta == nil {
		if seq > s.latest.Seq {
			return nil, errStale("unknown future frame_seq")
		}
		return nil, errStale("frame_seq is behind the live stream or expired")
	}
	age := time.Since(meta.CapturedAt)
	if meta.CapturedAt.IsZero() {
		age = s.latest.Age()
	}
	if maxAge > 0 && age > maxAge {
		return nil, errStale("frame too old")
	}
	if s.latest.Hash != "" && meta.Hash != "" && s.latest.Hash != meta.Hash {
		return nil, errStale("screen changed since frame_seq")
	}
	return meta, nil
}

func errStale(msg string) error {
	return staleError{msg: msg}
}

type staleError struct {
	msg string
}

func (e staleError) Error() string { return e.msg }
