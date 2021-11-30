package main

import (
	direct "github.com/buger/goterm"
	"math/rand"
	"sync/atomic"
	"time"
)

var monotonicId int64

func newThrottle(every time.Duration, done bool) *throttle {
	var left time.Duration = 0

	if !done {
		left = every
	}

	return &throttle{
		left: left,
		duration: every,
	}
}

type throttle struct {
	left, duration time.Duration
}

func (t *throttle) Reach(timeLeft time.Duration) bool {
	if t.left < t.duration {
		t.left += timeLeft
		return false
	}
	t.left = 0
	return true
}

func (t *throttle) Reset() {
	t.left = 0
}

func (t *throttle) Copy() *throttle {
	copy := *t
	return &copy
}

func newRandomCoordinate() Point {

	w := direct.Width()
	h := direct.Height()
	if w <= 0 {
		w = 100
	}
	if h <= 0 {
		h = 100
	}

	return Point{
		X: float64(rand.Intn(w)),
		Y: float64(rand.Intn(h)),
	}
}

func genId() int64 {
	old := atomic.LoadInt64(&monotonicId)
	swapped := atomic.CompareAndSwapInt64(&monotonicId, old, old+1)
	for !swapped {
		old = atomic.LoadInt64(&monotonicId)
		swapped = atomic.CompareAndSwapInt64(&monotonicId, old, old+1)
	}
	return old + 1
}
