package collider

import (
	"github.com/tanema/ump"
)

const COLLISION_AT_LEFT = 0x1
const COLLISION_AT_RIGHT = 0x2
const COLLISION_AT_TOP = 0x5
const COLLISION_AT_BOTTOM = 0x8
const COLLISION_IMPACT = 0x10

type CollisionInfo struct {
	Object  Collideable
	Details *ump.Collision
}

type ClBody struct {
	x, y, w, h        float64
	static, penetrate bool
	realBody          *ump.Body
	collisionInfo     *CollisionInfoSet
	ver               bool //odd even
	First, Next       *ClBody
	filter            string
}

func (receiver *ClBody) CollisionInfo() *CollisionInfoSet {
	return receiver.collisionInfo
}

func (receiver *ClBody) Collided() bool {
	if receiver.collisionInfo == nil {
		return false
	}
	return receiver.collisionInfo.Size() > 0
}

func (receiver *ClBody) RelativeMove(x, y float64) {
	receiver.x += x
	receiver.y += y
}

func (receiver *ClBody) Move(x, y float64) {
	receiver.x = x
	receiver.y = y
}

//after collision correction
func (receiver *ClBody) Correct(x, y float64) {
	receiver.x = x
	receiver.y = y
}

func (receiver *ClBody) Resize(w, h float64) {
	if receiver.realBody != nil {
		if receiver.w != w || receiver.h != h {
			panic("unable dynamic resize clBody")
		}
	}
	receiver.w = w
	receiver.h = h
}

func (receiver *ClBody) GetXY() (x float64, y float64) {
	return receiver.x, receiver.y
}

func (receiver *ClBody) GetWH() (w float64, h float64) {
	return receiver.w, receiver.h
}

func (receiver *ClBody) GetRect() (x float64, y float64, w float64, h float64) {
	return receiver.x, receiver.y, receiver.w, receiver.h
}

func (receiver *ClBody) GetCenter() (float64, float64) {
	return receiver.x + receiver.w/2, receiver.y + receiver.h/2
}

func (receiver *ClBody) Copy() *ClBody {
	instance := *receiver
	instance.realBody = nil
	instance.collisionInfo = nil //todo fix init
	instance.First = &instance
	if receiver.Next != nil {
		instance.Next = receiver.Next.Copy()
	}
	return &instance
}

func NewCollision(x, y, w, h float64) *ClBody {
	body := &ClBody{
		x: x, y: y, w: w, h: h,
	}
	body.First = body
	body.static = false
	body.penetrate = false
	body.filter = "base"
	return body
}

func NewStaticCollision(x, y, w, h float64) *ClBody {
	body := &ClBody{
		x: x, y: y, w: w, h: h,
	}
	body.First = body
	body.static = true
	body.penetrate = false
	body.filter = "static"
	return body
}

func NewPenetrateCollision(x, y, w, h float64) *ClBody {
	body := &ClBody{
		x: x, y: y, w: w, h: h,
	}
	body.First = body
	body.static = false
	body.penetrate = true
	body.filter = "penetrate"
	return body
}

//no cycle detection!
func LinkClBody(first, second *ClBody) {
	first.First = first
	second.First = first
	for true {
		if first.Next == nil {
			first.Next = second
			break
		}
		first = first.Next
	}
}
