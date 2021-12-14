package main

import (
	direct "github.com/buger/goterm"
	"math"
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
		left:     left,
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

func absMax(arguments ...float64) float64 {
	var cur float64
	var curi int
	for i := 0; i < len(arguments); i++ {
		if math.Abs(arguments[i]) > cur {
			cur = arguments[i]
			curi = i
		}
	}
	return arguments[curi]
}

func absMin(arguments ...float64) float64 {
	var cur float64
	var curi int
	for i := 0; i < len(arguments); i++ {
		if math.Abs(arguments[i]) < cur {
			cur = arguments[i]
			curi = i
		}
	}
	return arguments[curi]
}

func getDistance(x1, y1, x2, y2 float64) float64 {
	distX, distY := x1-x2, y1-y2
	return distX*distX + distY*distY
}

func divRemF(numenator, denumenator float64) float64 {
	return numenator - float64(int(math.Round(numenator/denumenator)))*denumenator
}

func triang(x int64) int64 {
	return int64(float64(x) / float64(2) * float64(x+1))
}

//rand betwen [1, max] with probability of n
func triangRand(max int64) int64 {
	rand := int64(rand.Intn(int(triang(max)))) + 1
	for i := int64(1); i <= max; i++ {
		tri := triang(i)
		if rand <= tri {
			return i
		}
	}
	return 0
}

func absInt64(n int64) int64 {
	y := n >> 63
	return (n ^ y) - y
}

//no special cases check :(
func absInt(n int) int {
	if n > 0 {
		return n
	} else {
		return n * -1
	}
}

// no special cases
func maxInt64(x, y int64) int64 {
	if x > y {
		return x
	}
	return y
}

func maxInt(x, y int) int {
	if x > y {
		return x
	}
	return y
}

func minInt(x, y int) int {
	if x < y {
		return x
	}
	return y
}

func IsInf(f int64, sign int) bool {
	// Test for infinity by comparing against maximum float.
	// To avoid the floating-point hardware, could use:
	//	x := Float64bits(f);
	//	return sign >= 0 && x == uvinf || sign <= 0 && x == uvneginf;
	return sign >= 0 && f > math.MaxInt64 || sign <= 0 && f < -math.MaxInt64
}
