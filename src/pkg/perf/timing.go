package perf

import "time"

// Timing is a lightweight stopwatch for segmented latency.
type Timing struct {
	start time.Time
	mark  time.Time
}

// Start returns a timing that begins now.
func Start() Timing {
	now := time.Now()
	return Timing{start: now, mark: now}
}

// Lap returns milliseconds since the previous lap (or start) and advances the mark.
func (t *Timing) Lap() int64 {
	now := time.Now()
	ms := now.Sub(t.mark).Milliseconds()
	t.mark = now
	return ms
}

// Total returns milliseconds since start.
func (t Timing) Total() int64 {
	return time.Since(t.start).Milliseconds()
}

// Percentile returns the nearest-rank percentile for a sorted-or-unsorted sample.
// p is in 0–100. An empty sample returns 0.
func Percentile(samples []int64, p float64) int64 {
	if len(samples) == 0 {
		return 0
	}
	cp := append([]int64(nil), samples...)
	for i := 1; i < len(cp); i++ {
		v := cp[i]
		j := i - 1
		for j >= 0 && cp[j] > v {
			cp[j+1] = cp[j]
			j--
		}
		cp[j+1] = v
	}
	if p <= 0 {
		return cp[0]
	}
	if p >= 100 {
		return cp[len(cp)-1]
	}
	idx := int((p / 100) * float64(len(cp)-1))
	return cp[idx]
}

// Max returns the largest sample, or 0 if empty.
func Max(samples []int64) int64 {
	if len(samples) == 0 {
		return 0
	}
	max := samples[0]
	for _, v := range samples[1:] {
		if v > max {
			max = v
		}
	}
	return max
}
