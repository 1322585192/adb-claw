package server

import (
	"sync"
	"time"

	"github.com/llm-net/adb-claw/pkg/observe"
)

// Session keeps the latest versioned UI snapshot for act() without re-dump.
type Session struct {
	mu     sync.Mutex
	latest *observe.Snapshot
}

func (s *Session) Put(snap *observe.Snapshot) {
	if snap == nil {
		return
	}
	s.mu.Lock()
	s.latest = snap
	s.mu.Unlock()
}

func (s *Session) Get(stateID string, maxAge time.Duration) (*observe.Snapshot, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.latest == nil {
		return nil, errStale("no cached UI snapshot; call observe first")
	}
	if maxAge > 0 && time.Since(s.latest.CapturedAt) > maxAge {
		return nil, errStale("cached UI snapshot expired; re-observe before acting")
	}
	if stateID != "" && s.latest.StateID != stateID {
		return nil, errStale("stale state_id " + stateID + " (latest is " + s.latest.StateID + ")")
	}
	return s.latest, nil
}

func errStale(msg string) error {
	return staleError{msg: msg}
}

type staleError struct {
	msg string
}

func (e staleError) Error() string { return e.msg }
