package main

import (
	"GoConsoleBT/collider"
	"errors"
	"github.com/tanema/ump"
	"time"
)

const OBJECT_EVENT_DESTROY = 1
const OBJECT_EVENT_DESPAWN = 2

const OBJECT_EVENT_RESET = 3
const OBJECT_EVENT_SPAWN = 4

var DestroyEvent Event = Event{
	EType:   OBJECT_EVENT_DESTROY,
	Payload: nil,
}

var DeSpawnEvent Event = Event{
	EType:   OBJECT_EVENT_DESPAWN,
	Payload: nil,
}

var ResetEvent Event = Event{
	EType:   OBJECT_EVENT_RESET,
	Payload: nil,
}

var SpawnEvent Event = Event{
	EType:   OBJECT_EVENT_SPAWN,
	Payload: nil,
}

var Tag404Error = errors.New("tag not found")

type Located interface {
	GetXY() (x float64, y float64)
}

type Sized interface {
	GetWH() (w float64, h float64)
}

type Scored interface {
	GetScore() int
}

type Danger interface {
	GetDamage(target Vulnerable) (value int, nemesis ObjectInterface)
	HasTag(tag string) bool
}

type Vulnerable interface {
	ReciveDamage(incoming Danger)
	HasTag(tag string) bool
}

type Appearable interface {
	GetAppearDuration() time.Duration
}

type Disappearable interface {
	GetDisappearDuration() time.Duration
}

type Obstacle interface {
}

type Prototyped interface {
	GetPrototype() ObjectInterface
}

type BlueprintMaked interface {
	GetBlueprint() string
}

type ObjectInterface interface {
	BlueprintMaked
	Prototyped
	Located
	Sized
	Update(timeLeft time.Duration) error
	GetSprite() Spriteer
	GetClBody() *collider.ClBody
	GetAttr() *Attributes
	GetTagValue(tag string, key string, defaultValue string) (string, error)
	GetOwner() ObjectInterface
	HasTag(tag string) bool
	Destroy(nemesis ObjectInterface) error //nemesis may be nil
	Reset() error                          //todo reset by configuration
	DeSpawn() error
	Spawn() error
}

type Point struct {
	X, Y float64
}

type Object struct {
	*Attributes
	*collider.Interactions
	destroyed, spawned bool
	blueprint          string
	sprite             Spriteer
	collision          *collider.ClBody
	zIndex             int
	tag                []string
	tagValues          map[string]*TagValue
	spawnCount         int64
	Owner, Prototype   ObjectInterface
}

func (receiver *Object) Update(timeLeft time.Duration) error {
	if receiver.GetClBody() == nil || receiver.destroyed {
		return nil
	}
	receiver.Interactions.Interact(receiver, timeLeft)
	return nil
}

func (receiver *Object) OnTickCollide(object collider.Collideable, collision *ump.Collision) {
	logger.Println("warning: empty tick collide")
}

func (receiver *Object) OnStartCollide(object collider.Collideable, collision *ump.Collision) {
	logger.Println("warning: empty start collide")
}

func (receiver *Object) OnStopCollide(object collider.Collideable, duration time.Duration) {
	logger.Println("warning: empty stop collide")
}

func (receiver *Object) GetSprite() Spriteer {
	return receiver.sprite
}

func (receiver *Object) GetClBody() *collider.ClBody {
	return receiver.collision
}

func (receiver *Object) GetXY() (x, y float64) {
	return receiver.collision.GetXY()
}

func (receiver *Object) GetAttr() *Attributes {
	return receiver.Attributes
}

func (receiver *Object) GetWH() (x, y float64) {
	return receiver.collision.GetWH()
}

func (receiver *Object) GetBlueprint() string {
	return receiver.blueprint
}

func (receiver *Object) GetZIndex() int {
	return receiver.zIndex
}

func (receiver *Object) Destroy(nemesis ObjectInterface) error {
	receiver.destroyed = true
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
	return nil
}

func (receiver *Object) DeSpawn() error {
	SwitchSprite(nil, receiver.sprite)
	receiver.spawned = false
	return nil
}

func (receiver *Object) GetOwner() ObjectInterface {
	if receiver.Owner == nil {
		return receiver
	}
	return receiver.Owner
}

func (receiver *Object) GetPrototype() ObjectInterface {
	if receiver.Prototype == nil {
		return receiver
	}
	return receiver.Prototype
}

func (receiver *Object) addTag(tags ...string) {
	for _, tag := range tags {
		receiver.tag = append(receiver.tag, tag)
	}
}

func (receiver *Object) HasTag(tag string) bool {

	/*	switch receiver.GetAttr() {
		id
		team
		blueprint
		player
		obstacle
		danger
		vulnerable
		motioner
		evented
		controled
		collided
		renderable
		Tagable
	}*/

	for _, part := range receiver.tag {
		if part == tag {
			return true
		}
	}
	return false
}

func (receiver *Object) clearTags() {
	receiver.tag = receiver.tag[0:0]
	for index, _ := range receiver.tagValues {
		delete(receiver.tagValues, index)
	}
}

func (receiver *Object) removeTag(tag string) {
	for i, part := range receiver.tag {
		if part == tag {
			receiver.tag[i] = ""
		}
	}
	delete(receiver.tagValues, tag)
}

func (receiver *Object) GetTag(tag string, makeIfNil bool) (*TagValue, error) {
	if _, ok := receiver.tagValues[tag]; !ok {
		if makeIfNil {
			receiver.tagValues[tag], _ = NewTagValue()
		} else {
			return nil, Tag404Error
		}
	}
	return receiver.tagValues[tag], nil
}

/**
* get tag value without allocation
 */
func (receiver *Object) GetTagValue(tag string, key string, defaultValue string) (string, error) {
	if tag, ok := receiver.tagValues[tag]; ok {
		return tag.Get(key, defaultValue), nil
	}
	return defaultValue, Tag404Error
}

func (receiver *Object) Reset() error {
	receiver.destroyed = false
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
	if receiver.tag != nil {
		instance.tag = make([]string, len(receiver.tag), cap(receiver.tag))
		copy(instance.tag, receiver.tag)
	}
	if receiver.tagValues != nil {
		instance.tagValues = make(map[string]*TagValue, len(receiver.tagValues))
		for index, value := range receiver.tagValues {
			instance.tagValues[index] = value
		}
	}
	return &instance
}

func NewObject(s Spriteer, c *collider.ClBody) (*Object, error) {
	interactions, _ := collider.NewIteractions()
	return &Object{
		Attributes:   new(Attributes),
		Interactions: interactions,
		destroyed:    false,
		spawned:      false,
		blueprint:    "",
		sprite:       s,
		collision:    c,
		zIndex:       0,
		tag:          nil,
		tagValues:    make(map[string]*TagValue),
		spawnCount:   0,
		Owner:        nil,
		Prototype:    nil,
	}, nil
}
