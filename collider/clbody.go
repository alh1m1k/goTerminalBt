package collider

import (
	"errors"
	"github.com/alh1m1k/ump"
)

var (
	CollisionNotFoundError = errors.New("collision not found")
)

type CollisionInfo struct {
	Object  Collideable
	Details *ump.Collision
}

// todo tags && hasTag
type ClBody struct {
	x, y, w, h              float64
	static, penetrate, fake bool
	realBody                *ump.Body
	collisionInfo           *CollisionInfoSet
	ver                     bool //odd even
	First, Next             *ClBody
	filter                  string
}

func (receiver *ClBody) CollisionInfo() *CollisionInfoSet {
	return receiver.collisionInfo
}

func (receiver *ClBody) Collided() bool {
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

// after collision correction
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

func (receiver *ClBody) FindExact(col *ump.Collision) (*ClBody, error) {
	curr := receiver.First
	for {
		if curr == nil {
			return nil, CollisionNotFoundError
		}
		if curr.realBody == col.Body {
			return curr, nil
		}
		curr = curr.Next
	}
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

func (receiver *ClBody) SetStatic(value bool) {
	receiver.static = value
}

func (receiver *ClBody) IsStatic() bool {
	return receiver.static
}

func (receiver *ClBody) IsPenetrate() bool {
	return receiver.penetrate
}

func (receiver *ClBody) IsFake() bool {
	return receiver.fake
}

func (receiver *ClBody) Copy() *ClBody {
	instance := *receiver
	instance.realBody = nil
	instance.collisionInfo = NewCollisionInfo(len(instance.collisionInfo.m))
	instance.First = &instance
	if receiver.Next != nil {
		instance.Next = receiver.Next.Copy()
	}
	return &instance
}

func (receiver *ClBody) HasTag(tag string) bool {
	return false
}

// interface recursion
func (receiver *ClBody) GetClBody() *ClBody {
	return receiver
}

func NewFakeCollision(x, y, w, h float64) *ClBody {
	body := &ClBody{
		x: x, y: y, w: w, h: h,
	}
	body.First = body
	body.static = true
	body.penetrate = true
	body.fake = true
	body.filter = ""
	body.collisionInfo = NewCollisionInfo(5)
	return body
}

func NewCollision(x, y, w, h float64) *ClBody {
	body := &ClBody{
		x: x, y: y, w: w, h: h,
	}
	body.First = body
	body.static = false
	body.penetrate = false
	body.filter = "base"
	body.collisionInfo = NewCollisionInfo(5)
	return body
}

func NewPerimeterCollision(x, y, w, h float64) *ClBody {
	body := &ClBody{
		x: x, y: y, w: w, h: h,
	}
	body.First = body
	body.static = false
	body.penetrate = false
	body.filter = "perimeter"
	body.collisionInfo = NewCollisionInfo(5)
	return body
}

/*
func NewVisionCollision(x, y, w, h float64) *ClBody {
	body := &ClBody{
		x: x, y: y, w: w, h: h,
	}
	body.First 		= body
	body.static 	= false
	body.penetrate 	= false
	body.vision 	= true
	body.filter 	= "vision"
	return body
}*/

func NewStaticCollision(x, y, w, h float64) *ClBody {
	body := &ClBody{
		x: x, y: y, w: w, h: h,
	}
	body.First = body
	body.static = true
	body.penetrate = false
	body.filter = "static"
	body.collisionInfo = NewCollisionInfo(5)
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
	body.collisionInfo = NewCollisionInfo(5)
	return body
}

// no cycle detection!
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
