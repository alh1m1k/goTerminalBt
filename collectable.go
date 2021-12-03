package main

import (
	"GoConsoleBT/collider"
	"github.com/tanema/ump"
	"time"
)

const COLLECT_EVENT_COLLECTED = 400

var CollectEvent Event = Event{
	EType:   COLLECT_EVENT_COLLECTED,
	Payload: nil,
}

type Collectable struct {
	*Object
	*ObservableObject
	*throttle
	*State
	Owner  ObjectInterface
	Damage int
	Ttl    time.Duration
}

func (receiver *Collectable) ApplyState(current *StateItem) error {
	SwitchSprite(current.StateInfo.(*UnitStateInfo).sprite, receiver.sprite)
	receiver.sprite = current.StateInfo.(*UnitStateInfo).sprite
	return nil
}

func (receiver *Collectable) Update(timeLeft time.Duration) error {
	if receiver.destroyed {
		return nil
	}

	receiver.Object.Update(timeLeft)

	if receiver.throttle != nil && receiver.throttle.Reach(timeLeft) {
		receiver.Destroy(nil)
	}

	return nil
}

func (receiver *Collectable) OnTickCollide(object collider.Collideable, collision *ump.Collision) {

}

func (receiver *Collectable) OnStartCollide(object collider.Collideable, collision *ump.Collision) {
	if object.HasTag("tank") {
		receiver.Collect(object.(*Unit))
	}
}

func (receiver *Collectable) OnStopCollide(object collider.Collideable, duration time.Duration) {

}

func (receiver *Collectable) Collect(by *Unit) error {
	if receiver.destroyed {
		return nil
	}
	logger.Printf("collected by %d cycleId %d", by.ID, CycleID)
	receiver.Trigger(CollectEvent, receiver, by)
	//todo enteer state collected
	receiver.Destroy(nil)
	return nil
}

func (receiver *Collectable) GetAppearDuration() time.Duration {
	return 3 * time.Second
}

func (receiver *Collectable) Destroy(nemesis ObjectInterface) error {
	if receiver.destroyed {
		return nil
	}
	receiver.Object.Destroy(nemesis)
	receiver.Trigger(DestroyEvent, receiver, nil)
	return nil
}

func (receiver *Collectable) Reset() error {
	receiver.Object.Reset()
	receiver.State.Reset()
	if receiver.Ttl > 0 {
		receiver.throttle = newThrottle(receiver.Ttl, false)
	}
	if receiver.throttle != nil {
		receiver.throttle.Reset()
	}
	receiver.Trigger(ResetEvent, receiver, nil)
	return nil
}

func (receiver *Collectable) Spawn() error {
	receiver.Object.Spawn()
	receiver.Trigger(SpawnEvent, receiver, nil)

	return nil
}

func (receiver *Collectable) DeSpawn() error {
	receiver.Trigger(DeSpawnEvent, receiver, nil)
	receiver.Object.DeSpawn()
	return nil
}

func (receiver *Collectable) Copy() *Collectable {
	instance := *receiver

	instance.ObservableObject = receiver.ObservableObject.Copy()
	instance.ObservableObject.Owner = instance
	instance.Object = receiver.Object.Copy()
	instance.State = receiver.State.Copy()
	instance.State.Owner = &instance
	instance.Interactions.Subscribe(&instance)
	if instance.throttle != nil {
		instance.throttle = receiver.throttle.Copy()
	}

	return &instance
}

func NewCollectable2(obj *Object, oo *ObservableObject, state *State, Owner ObjectInterface) (*Collectable, error) {
	instance := &Collectable{
		Object:           obj,
		ObservableObject: oo,
		State:            state,
		Owner:            Owner,
	}
	instance.ObservableObject.Owner = instance
	instance.State.Owner = instance
	instance.Interactions.Subscribe(instance)
	return instance, nil
}
