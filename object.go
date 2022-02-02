package main

import (
	"GoConsoleBT/collider"
	"errors"
	"github.com/alh1m1k/ump"
	"time"
)

const OBJECT_EVENT_DESTROY = 1
const OBJECT_EVENT_DESPAWN = 2
const OBJECT_EVENT_RESET = 3
const OBJECT_EVENT_SPAWN = 4

var (
	DestroyEvent Event = Event{
		EType:   OBJECT_EVENT_DESTROY,
		Payload: nil,
	}
	DeSpawnEvent Event = Event{
		EType:   OBJECT_EVENT_DESPAWN,
		Payload: nil,
	}
	ResetEvent Event = Event{
		EType:   OBJECT_EVENT_RESET,
		Payload: nil,
	}
	SpawnEvent Event = Event{
		EType:   OBJECT_EVENT_SPAWN,
		Payload: nil,
	}

	Tag404Error = errors.New("tag not found")
)

type Scored interface {
	GetScore() int
}

type Obstacle interface {
}

type ObjectInterface interface {
	Located
	Sized
	Updateable
	Renderable
	Tagable
	Seen
	Trackable
	collider.Collideable
	GetCenter() Center
	GetTracker() *Tracker
	GetAttr() *Attributes
	GetOwner() ObjectInterface
	Destroy(nemesis ObjectInterface) error //nemesis may be nil
	Reset() error                          //todo reset by configuration
	DeSpawn() error
	Spawn() error
	Move(x, y float64)
	RelativeMove(x, y float64)
}

type Object struct {
	*Attributes
	collision *collider.ClBody
	*collider.Interactions
	sprite Spriteer
	*Tracker
	*Tags
	Owner, Prototype   ObjectInterface
	destroyed, spawned bool
	blueprint          string
	zIndex             int
	spawnCount         int64
}

func (receiver *Object) Update(timeLeft time.Duration) error {
	if receiver.GetClBody() == nil || receiver.destroyed {
		return nil
	}
	receiver.Interactions.Interact(receiver, timeLeft)
	if receiver.Tracker != nil {
		receiver.Tracker.Update(receiver.GetXY(), receiver.GetWH())
	}
	return nil
}

func (receiver *Object) OnTickCollide(object collider.Collideable, collision *ump.Collision, owner *collider.Interactions) {
	logger.Println("warning: empty tick collide")
}

func (receiver *Object) OnStartCollide(object collider.Collideable, collision *ump.Collision, owner *collider.Interactions) {
	logger.Println("warning: empty start collide")
}

func (receiver *Object) OnStopCollide(object collider.Collideable, duration time.Duration, owner *collider.Interactions) {
	logger.Println("warning: empty stop collide")
}

func (receiver *Object) GetSprite() Spriteer {
	return receiver.sprite
}

func (receiver *Object) GetClBody() *collider.ClBody {
	return receiver.collision
}

func (receiver *Object) Move(x, y float64) {
	receiver.collision.Move(x, y)
}

func (receiver *Object) RelativeMove(x, y float64) {
	receiver.collision.RelativeMove(x, y)
}

func (receiver *Object) GetXY() Point {
	x, y := receiver.collision.GetXY()
	return Point{x, y}
}

func (receiver *Object) GetWH() Size {
	w, h := receiver.collision.GetWH()
	return Size{
		W: w,
		H: h,
	}
}

func (receiver *Object) GetCenter() Center {
	x, y := receiver.collision.GetCenter()
	return Center{
		X: x,
		Y: y,
	}
}

func (receiver *Object) GetRect() (x, y, w, h float64) {
	return receiver.collision.GetRect()
}

func (receiver *Object) GetTracker() *Tracker {
	return receiver.Tracker
}

func (receiver *Object) GetAttr() *Attributes {
	return receiver.Attributes
}

func (receiver *Object) GetLayer() int {
	return receiver.layer
}

func (receiver *Object) GetZIndex() int {
	return receiver.zIndex
}

func (receiver *Object) Destroy(nemesis ObjectInterface) error {
	receiver.destroyed = true
	receiver.Attributes.Destroyed = true
	return nil
}

func (receiver *Object) Spawn() error {
	if receiver.spawnCount > 0 {
		SwitchSprite(receiver.sprite, receiver.sprite)
	} else {
		SwitchSprite(receiver.sprite, nil)
	}
	receiver.spawned = true
	receiver.spawnCount++
	receiver.Attributes.Spawned = true
	return nil
}

func (receiver *Object) DeSpawn() error {
	SwitchSprite(nil, receiver.sprite)
	receiver.spawned = false
	receiver.Attributes.Spawned = false
	return nil
}

func (receiver *Object) GetVision() *collider.ClBody {
	return nil
}

func (receiver *Object) GetOwner() ObjectInterface {
	if receiver.Owner == nil {
		return receiver
	}
	return receiver.Owner
}

func (receiver *Object) Reset() error {
	receiver.destroyed = false
	receiver.Attributes.Destroyed = false
	receiver.Interactions.Clear()
	return nil
}

func (receiver *Object) Free() {
	receiver.clearTags()
}

func (receiver *Object) Copy() *Object {
	instance := *receiver
	attributes := *receiver.Attributes //todo same?
	instance.Attributes = &attributes
	instance.sprite = CopySprite(receiver.sprite)
	instance.collision = receiver.collision.Copy()
	instance.Interactions = receiver.Interactions.Copy()
	if receiver.Tracker != nil {
		instance.Tracker = receiver.Tracker.Copy()
	}
	if receiver.Tags != nil {
		instance.Tags = receiver.Tags.Copy()
	}
	return &instance
}

func NewObject(s Spriteer, c *collider.ClBody) (*Object, error) {
	interactions, _ := collider.NewIteractions()
	tags, _ := NewTags()
	return &Object{
		Attributes:   new(Attributes),
		Interactions: interactions,
		destroyed:    false,
		spawned:      false,
		blueprint:    "",
		sprite:       s,
		collision:    c,
		zIndex:       0,
		Tags:         tags,
		spawnCount:   0,
		Owner:        nil,
		Prototype:    nil,
	}, nil
}

func GetObjectState(id string) (*State, error) {
	return GetState(id, func(m map[string]interface{}) (interface{}, error) {
		var sprite Spriteer = nil
		var err error

		if animation, ok := m["animation"]; ok {
			//todo refactor this shit
			animationInfo := animation.(map[string]interface{})
			sprite, _ = GetAnimation(animationInfo["name"].(string), int(animationInfo["length"].(float64)), true, false)
			if sprite != nil {
				spriteAsAnimation := sprite.(*Animation)
				spriteAsAnimation.Cycled = animationInfo["cycled"].(bool)
				spriteAsAnimation.Duration = time.Duration(animationInfo["duration"].(float64))
				if blink, ok := animationInfo["blink"]; ok {
					spriteAsAnimation.BlinkRate = time.Duration(blink.(float64))
				}
			}
		}
		if sprite == nil {
			sprite, err = GetSprite(m["sprite"].(string), true, false)
			if err != nil {
				return nil, err
			}
		}

		return &UnitStateInfo{
			sprite:     sprite,
			collisionX: 0,
			collisionY: 0,
			collisionW: 0,
			collisionH: 0,
		}, nil
	})
}
