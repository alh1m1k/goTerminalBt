package main

import (
	"errors"
	"sync"
	"sync/atomic"
	"time"
)

var ReloadError = errors.New("gun on reload")
var GunConfigError = errors.New("gun not configurated")
var OutAmmoError = errors.New("out of ammo")

type FireParams struct {
	Position, Direction, BaseSpeed Point
	Owner                          ObjectInterface
}

type GunState struct {
	Projectile       string
	Name             string
	Ammo             int64
	ShotQueue        int
	PerShotQueueTime time.Duration
	ReloadTime       time.Duration
	lastShotTime     time.Time
}

type Gun struct {
	Owner   *Unit
	Current *GunState
	State   []*GunState
	mutex   sync.Mutex
}

func (receiver *Gun) Fire() error {
	receiver.mutex.Lock()
	var current = receiver.Current
	if current == nil {
		receiver.mutex.Unlock()
		return GunConfigError
	}
	var delayAccumulator time.Duration
	if receiver.IsReloading() {
		receiver.mutex.Unlock()
		return ReloadError
	}
	current.lastShotTime = time.Now()
	receiver.mutex.Unlock()
	params := receiver.getParams()
	for i := 0; i < current.ShotQueue; i++ {
		if current.Ammo != -1 && current.Ammo <= 0 {
			return OutAmmoError
		}
		if current.PerShotQueueTime > 0 && i > 1 {
			delayAccumulator += current.PerShotQueueTime
			time.AfterFunc(delayAccumulator, func() {
				if receiver.Owner.destroyed {
					return
				}
				receiver.Owner.Trigger(FireEvent, receiver.Owner, params)
				current.lastShotTime = time.Now()
				if current.Ammo > 0 {
					current.Ammo--
				}
			})

		} else {
			receiver.Owner.Trigger(FireEvent, receiver.Owner, params)
			current.lastShotTime = time.Now()
			if current.Ammo > 0 {
				current.Ammo--
			}
		}
	}
	return nil
}

func (receiver *Gun) IsReloading() bool {
	if receiver.Current.lastShotTime.IsZero() {
		return false
	}
	if time.Now().Sub(receiver.Current.lastShotTime) > receiver.Current.ReloadTime {
		return false
	}
	return true
}

func (receiver *Gun) GetProjectile() string {
	if receiver.Current == nil {
		return ""
	}
	return receiver.Current.Projectile
}

func (receiver *Gun) GetName() string {
	if receiver.Current.Name != "" {
		return receiver.Current.Name
	}
	return "N/A"
}

func (receiver *Gun) IncAmmoIfAcceptable(byValue int64) int64 {
	for {
		//what if receiver.Current change???
		if ref := atomic.LoadInt64(&receiver.Current.Ammo); ref > -1 {
			if !atomic.CompareAndSwapInt64(&receiver.Current.Ammo, ref, ref+byValue) {
				continue
			} else {
				return atomic.LoadInt64(&receiver.Current.Ammo)
			}
		} else {
			return -1
		}
	}
}

func (receiver *Gun) Reset() {
	receiver.mutex.Lock()
	defer receiver.mutex.Unlock()
	receiver.Current = receiver.State[0]
}

func (receiver *Gun) Downgrade() {
	receiver.mutex.Lock()
	defer receiver.mutex.Unlock()
	receiver.Current = receiver.State[0]
}

func (receiver *Gun) Upgrade(state *GunState) {
	receiver.mutex.Lock()
	defer receiver.mutex.Unlock()
	receiver.State[1] = state
	receiver.Current = state
}

func (receiver *Gun) Basic(state *GunState) {
	receiver.mutex.Lock()
	defer receiver.mutex.Unlock()
	receiver.State[0] = state
	receiver.Current = state
}

func (receiver *Gun) getPosition() Point {
	x, y := receiver.Owner.GetXY()
	dir := receiver.Owner.Direction
	w, h := receiver.Owner.GetWH()

	if dir.X == 0 && dir.Y == 0 {
		dir.Y = -1
	}
	centerX := x + w/2
	centerY := y + h/2

	return Point{
		X: centerX + (dir.X * w / 2),
		Y: centerY + (dir.Y * h / 2),
	}
}

func (receiver *Gun) getParams() FireParams {
	return FireParams{
		Position:  receiver.getPosition(),
		BaseSpeed: receiver.Owner.Speed,
		Direction: receiver.Owner.Direction,
		Owner:     receiver.Owner,
	}
}

func (receiver *Gun) Copy() *Gun {
	instance := *receiver
	instance.State = make([]*GunState, len(receiver.State), cap(receiver.State))
	for i, state := range receiver.State {
		if state == nil {
			continue
		}
		copy := *state
		instance.State[i] = &copy
		if state == receiver.Current {
			instance.Current = &copy
		}
	}
	instance.mutex = sync.Mutex{}
	return &instance
}

func NewGun(owner *Unit) (*Gun, error) {
	return &Gun{
		Owner:   owner,
		Current: nil,
		State:   make([]*GunState, 2, 2),
	}, nil
}
