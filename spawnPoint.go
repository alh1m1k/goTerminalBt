package main

import (
	"GoConsoleBT/collider"
	direct "github.com/buger/goterm"
	"sync/atomic"
	"time"
)

const SPAWN_POINT_STATUS = 501

var (
	SpawnPointStatus Event = Event{
		EType:   SPAWN_POINT_STATUS,
		Payload: nil,
	}
	SPSAvailableSprite   = NewContentSprite([]byte(direct.Color("Free", direct.GREEN)))
	SPSUnavailableSprite = NewContentSprite([]byte(direct.Color("Lock", direct.RED)))
)

type SpawnPoint struct {
	*Object
	*ObservableObject
	allowList [][]string
	crossing  bool
	capture   int64
}

func (receiver *SpawnPoint) AddAllowing(tag ...string) {
	receiver.allowList = append(receiver.allowList, tag)
}

func (receiver *SpawnPoint) CanSpawn(object Tagable) bool {
list:
	for _, andTag := range receiver.allowList {
		for _, tag := range andTag {
			if !object.HasTag(tag) {
				continue list
			}
		}
		return true
	}
	return false
}

func (receiver *SpawnPoint) IsAvailable() bool {
	return !receiver.crossing && receiver.capture == 0
}

func (receiver *SpawnPoint) Capture() bool {
	return atomic.CompareAndSwapInt64(&receiver.capture, 0, 1)
}

func (receiver *SpawnPoint) Update(timeLeft time.Duration) error {
	if clBody := receiver.GetClBody(); clBody != nil {
		crossing := false
		for opposite, _ := range clBody.CollisionInfo().I() {
			if !opposite.HasTag("obstacle") && !opposite.HasTag("spawnPoint") {
				continue
			}
			if oiOpposite, ok := opposite.(ObjectInterface); !ok {
				continue
			} else {
				dist := receiver.GetCenter().Minus(oiOpposite.GetCenter()).Abs()
				size := receiver.GetWH().Plus(oiOpposite.GetWH()).Divide(Size{2, 2})
				dist = dist.Minus(Center{size.W, size.H})
				if dist.X < -collider.GRID_COORD_TOLERANCE && dist.Y < -collider.GRID_COORD_TOLERANCE {
					crossing = true
					break
				}
			}
		}
		if crossing != receiver.crossing {
			receiver.crossing = crossing
			if crossing {
				receiver.removeTag("available")
				receiver.addTag("unavailable")
				if DEBUG_SPAWN_POINT_STATUS {
					receiver.sprite = SPSUnavailableSprite
					logger.Printf("cycle: %d spawn point %d is now unavailable\n", CycleID, receiver.ID)
				}
				receiver.Trigger(SpawnPointStatus, receiver, false)
			} else {
				receiver.removeTag("unavailable")
				receiver.addTag("available")
				if DEBUG_SPAWN_POINT_STATUS {
					receiver.sprite = SPSAvailableSprite
					logger.Printf("cycle: %d spawn point %d is now available\n", CycleID, receiver.ID)
				}
				receiver.Trigger(SpawnPointStatus, receiver, true)
				receiver.capture = 0
			}
		}
	}
	return nil
}

func (receiver *SpawnPoint) Copy() *SpawnPoint {
	instance := *receiver
	instance.Object = receiver.Object.Copy()
	instance.ObservableObject = receiver.ObservableObject.Copy()
	instance.ObservableObject.Owner = instance
	instance.allowList = make([][]string, len(receiver.allowList))
	for i, list := range receiver.allowList {
		instance.allowList[i] = make([]string, 0, len(list))
		for _, tag := range list {
			instance.allowList[i] = append(instance.allowList[i], tag)
		}
	}
	return &instance
}

func NewSpawnPoint(object *Object, oo *ObservableObject) (*SpawnPoint, error) {
	var sprite Spriteer
	inst := new(SpawnPoint)
	inst.Object = object
	inst.ObservableObject = oo
	oo.Owner = inst
	if DEBUG_SPAWN_POINT_STATUS {
		sprite = SPSAvailableSprite
		inst.Attributes.Renderable = true
	} else {
		sprite = NewContentSprite([]byte{})
		inst.Attributes.Renderable = false
	}
	if !inst.HasTag("spawnPoint") {
		inst.addTag("spawnPoint")
	}
	inst.sprite = sprite
	return inst, nil
}
