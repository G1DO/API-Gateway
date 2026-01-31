package ratelimit

import (
	"sync"
	"time"
)

// SlidingWindow implements the sliding window counter rate limiting algorithm.
//
// Instead of hard window boundaries (which allow 2x burst at edges),
// it uses a weighted combination of the previous and current window counts:
//
//	effective = prevCount × (1 - elapsed/windowSize) + currentCount
//
// This approximates a true sliding window using only two counters
// and a timestamp — constant memory regardless of request volume.
type SlidingWindow struct {
	mu          sync.Mutex
	maxRequests int
	windowSize  time.Duration
	windowStart time.Time // start of current window
	prevCount   int       // requests in previous window
	currCount   int       // requests in current window
}

// NewSlidingWindow creates a sliding window limiter allowing maxRequests
// per windowSize duration.
func NewSlidingWindow(maxRequests int, windowSize time.Duration) *SlidingWindow {
	return &SlidingWindow{
		maxRequests: maxRequests,
		windowSize:  windowSize,
		windowStart: time.Now(),
	}
}

// Allow returns true if the request is within the rate limit.
func (sw *SlidingWindow) Allow() (ok bool, retryAfter time.Duration) {
	sw.mu.Lock()
	defer sw.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(sw.windowStart)

	// Advance windows if needed
	if elapsed >= 2*sw.windowSize {
		// Been idle for 2+ windows — reset everything
		sw.prevCount = 0
		sw.currCount = 0
		sw.windowStart = now
		elapsed = 0
	} else if elapsed >= sw.windowSize {
		// Current window is done — rotate
		sw.prevCount = sw.currCount
		sw.currCount = 0
		sw.windowStart = sw.windowStart.Add(sw.windowSize)
		elapsed = now.Sub(sw.windowStart)
	}

	// Weighted count: previous window's contribution fades as we move
	// through the current window
	weight := 1.0 - elapsed.Seconds()/sw.windowSize.Seconds()
	if weight < 0 {
		weight = 0
	}
	effective := float64(sw.prevCount)*weight + float64(sw.currCount)

	if effective+1 > float64(sw.maxRequests) {
		// How long until enough of the previous window fades
		// to allow one more request
		remaining := sw.windowSize - elapsed
		return false, remaining
	}

	sw.currCount++
	return true, 0
}