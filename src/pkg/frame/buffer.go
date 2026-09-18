package frame

import (
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/llm-net/adb-claw/pkg/atomicfile"
)

// Buffer keeps only the newest frame (capacity 1). Incoming frames replace
// any unread previous frame so the host never queues latency.
type Buffer struct {
	mu       sync.Mutex
	cond     *sync.Cond
	latest   *Frame
	lastHash string
	dropped  uint64
}

// NewBuffer returns an empty capacity-1 frame buffer.
func NewBuffer() *Buffer {
	b := &Buffer{}
	b.cond = sync.NewCond(&b.mu)
	return b
}

// Put stores f as the latest frame. A previous unread frame is dropped.
// Identical JPEG hashes are stored but marked Duplicate.
func (b *Buffer) Put(f *Frame) {
	if f == nil {
		return
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.latest != nil {
		b.dropped++
	}
	if b.lastHash != "" && f.Hash == b.lastHash {
		f.Duplicate = true
	}
	if f.Hash != "" {
		b.lastHash = f.Hash
	}
	b.latest = f
	b.cond.Broadcast()
}

// Latest returns the current frame or nil.
func (b *Buffer) Latest() *Frame {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.latest
}

// Dropped returns how many unread frames were replaced.
func (b *Buffer) Dropped() uint64 {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.dropped
}

// WaitAfter blocks until a non-duplicate frame with Seq > seq and a different
// hash arrives, or until timeout. On timeout it returns the current latest frame.
func (b *Buffer) WaitAfter(seq uint32, hash string, timeout time.Duration) *Frame {
	deadline := time.Now().Add(timeout)
	b.mu.Lock()
	defer b.mu.Unlock()

	for {
		if f := b.latest; f != nil {
			changed := f.Seq > seq && !f.Duplicate && (hash == "" || f.Hash != hash)
			if changed {
				return f
			}
		}
		remain := time.Until(deadline)
		if remain <= 0 {
			return b.latest
		}
		timer := time.AfterFunc(remain, func() {
			b.mu.Lock()
			b.cond.Broadcast()
			b.mu.Unlock()
		})
		b.cond.Wait()
		timer.Stop()
	}
}

// WriteAtomic writes jpeg bytes to path via a sibling temp file + rename.
func WriteAtomic(path string, jpeg []byte) error {
	return atomicfile.Write(path, jpeg, 0644)
}

// DefaultLatestPath is the atomic latest.jpg used by serve.
func DefaultLatestPath() string {
	return filepath.Join(os.TempDir(), "adb-claw-latest.jpg")
}
