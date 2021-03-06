package main

import (
	"context"
	"errors"
	"fmt"
	direct "github.com/buger/goterm"
	"math"
	"math/rand"
	"sync/atomic"
	"time"
)

var (
	monotonicId   int64
	noDescription = struct {
		Name        string
		Description string
	}{Name: "N/A", Description: "N/A"}
)

type MinMaxCurr struct {
	MinMax
	Current float64
}

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

func isInf(f int64, sign int) bool {
	// Test for infinity by comparing against maximum float.
	// To avoid the floating-point hardware, could use:
	//	x := Float64bits(f);
	//	return sign >= 0 && x == uvinf || sign <= 0 && x == uvneginf;
	return sign >= 0 && f > math.MaxInt64 || sign <= 0 && f < -math.MaxInt64
}

func every(duration time.Duration, ctx context.Context) <-chan time.Time {
	output := make(chan time.Time)
	go func(timer chan time.Time, ctx context.Context) {
		innerTimer := time.NewTimer(duration)
		for {
			select {
			case timeLeft := <-innerTimer.C:
				timer <- timeLeft
				innerTimer.Reset(duration)
			case <-ctx.Done():
				return
			}
		}
	}(output, ctx)
	return output
}

func everyFunc(duration time.Duration, callback func(), ctx context.Context) {
	output := make(chan time.Time)
	go func(timer chan time.Time, ctx context.Context) {
		innerTimer := time.NewTimer(duration)
		for {
			select {
			case <-innerTimer.C:
				go callback()
				innerTimer.Reset(duration)
			case <-ctx.Done():
				return
			}
		}
	}(output, ctx)
}

func GetTags(object ObjectInterface) (*Tags, error) {
	switch object.(type) {
	case *Unit:
		return object.(*Unit).Tags, nil
	case *Wall:
		return object.(*Wall).Tags, nil
	case *Projectile:
		return object.(*Projectile).Tags, nil
	case *Explosion:
		return object.(*Explosion).Tags, nil
	case *Collectable:
		return object.(*Collectable).Tags, nil
	case *Object:
		return object.(*Object).Tags, nil
	case *MotionObject:
		return object.(*MotionObject).Tags, nil
	case *SpawnPoint:
		return object.(*SpawnPoint).Tags, nil
	default:
		return nil, errors.New(fmt.Sprintf("GetTags unknown object type %t", object))
	}
}

func getProjectilePlDescription(blueprint string) struct {
	Name        string
	Description string
} {
	info, err := Info(blueprint)
	if err != nil {
		return noDescription
	}
	if info.Type != "projectile" {
		return noDescription
	}
	return struct {
		Name,
		Description string
	}{Name: info.Name, Description: info.Description}
}
