package main

import (
	"time"
)


type Wall struct {
	Object
	*ObservableObject
	*State
	HP, FullHP, Score int
}

func (receiver *Wall) Update(timeLeft time.Duration) error {
	if receiver.destroyed {
		return nil
	}
	if receiver.collision.Collided() {
		for object, _ := range receiver.collision.CollisionInfo().I() {
			if object.HasTag("danger") {
				//logger.Println("Damage", cycle)
				receiver.ReciveDamage(object.(Danger))
			}
		}
	}
	return nil
}

func (receiver *Wall) ReciveDamage(incoming Danger) {
	damage, nemesis := incoming.GetDamage(receiver)
	receiver.HP -= damage
	if damage <= 0 {
		return
	}
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
	receiver.State.Reset()
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
	instance.ObservableObject.Owner	= instance
	instance.Object     	  = *receiver.Object.Copy()
	instance.State 			  = receiver.State.Copy()
	instance.State.Owner	  = &instance

	return &instance
}

func NewWall2(obj Object, state *State, obs *ObservableObject) (*Wall, error) {
	instance := new(Wall)
	instance.Object = obj
	instance.ObservableObject = obs
	instance.ObservableObject.Owner = instance
	instance.State = state
	instance.State.Owner = instance
	instance.destroyed = false
	instance.spawned = false

	return instance, nil
}