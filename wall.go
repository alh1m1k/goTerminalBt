package main

import (
	"GoConsoleBT/collider"
	"github.com/tanema/ump"
	"time"
)

type Wall struct {
	*Object
	*ObservableObject
	*State
	HP, FullHP, Score int
}

func (receiver *Wall) Update(timeLeft time.Duration) error {
	if receiver.destroyed {
		return nil
	}
	receiver.Object.Update(timeLeft)
	return nil
}

func (receiver *Wall) OnTickCollide(object collider.Collideable, collision *ump.Collision, owner *collider.Interactions) {

}

func (receiver *Wall) OnStartCollide(object collider.Collideable, collision *ump.Collision, owner *collider.Interactions) {
	if object.HasTag("danger") && receiver.HasTag("vulnerable") {
		receiver.ReciveDamage(object.(Danger))
	}
}

func (receiver *Wall) OnStopCollide(object collider.Collideable, duration time.Duration, owner *collider.Interactions) {

}

func (receiver *Wall) ReciveDamage(incoming Danger) {
	damage, nemesis := incoming.GetDamage(receiver)
	if damage <= 0 {
		return
	}
	receiver.HP -= damage
	if receiver.HP <= 0 {
		receiver.Destroy(nemesis)
	} else {
		receiver.Trigger(DamadgeEvent, receiver, damage)
	}
}

func (receiver *Wall) GetScore() int {
	return receiver.Score
}

func (receiver *Wall) Destroy(nemesis ObjectInterface) error {
	if receiver.destroyed {
		return nil
	} //collision in cycle may cause multiple destroy
	receiver.Object.Destroy(nemesis)
	receiver.Trigger(DestroyEvent, receiver, nemesis)
	return nil
}

func (receiver *Wall) Reset() error {
	receiver.HP = receiver.FullHP
	receiver.Object.Reset()
	if receiver.State != nil {
		receiver.State.Reset()
	}
	receiver.Trigger(ResetEvent, receiver, nil)
	return nil
}

func (receiver *Wall) DeSpawn() error {
	receiver.Object.DeSpawn()
	receiver.Trigger(DeSpawnEvent, receiver, nil)
	return nil
}

func (receiver *Wall) Spawn() error {
	receiver.Object.Spawn()
	receiver.Trigger(SpawnEvent, receiver, nil)

	return nil
}

func (receiver *Wall) GetDisappearDuration() time.Duration {
	return 1 * time.Second
}

func (receiver *Wall) ApplyState(current *StateItem) error {
	SwitchSprite(current.StateInfo.(*UnitStateInfo).sprite, receiver.sprite)
	receiver.sprite = current.StateInfo.(*UnitStateInfo).sprite
	return nil
}

func (receiver *Wall) Copy() *Wall {
	instance := *receiver

	instance.ObservableObject = receiver.ObservableObject.Copy()
	instance.ObservableObject.Owner = instance
	instance.Object = receiver.Object.Copy()
	if receiver.State != nil {
		instance.State = receiver.State.Copy()
		instance.State.Owner = &instance
	}
	instance.Interactions.Subscribe(&instance)

	return &instance
}

func NewWall(obj *Object, state *State, obs *ObservableObject) (*Wall, error) {
	instance := new(Wall)

	instance.Object = obj
	instance.ObservableObject = obs
	instance.ObservableObject.Owner = instance
	if state != nil {
		instance.State = state
		instance.State.Owner = instance
	}
	instance.Interactions.Subscribe(instance)

	instance.destroyed = false
	instance.spawned = false

	return instance, nil
}
