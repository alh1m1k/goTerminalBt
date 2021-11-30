package main

import (
	"time"
)

type Explosion struct {
	*Object
	*ObservableObject
	*throttle
	Owner	ObjectInterface
	Damage 	int
	Ttl 	time.Duration
}

func (receiver *Explosion) ApplyState(current *StateItem) error {
	receiver.sprite = current.StateInfo.(*UnitStateInfo).sprite
	return nil
}

func (receiver *Explosion) Update(timeLeft time.Duration) error {
	if receiver.destroyed {
		return nil
	}

	if receiver.throttle != nil && receiver.throttle.Reach(timeLeft) {
		receiver.Destroy(nil)
	}

	return nil
}

func (receiver *Explosion) Destroy(nemesis ObjectInterface) error {
	if receiver.destroyed {
		return nil
	}
	receiver.Object.Destroy(nemesis)
	receiver.Trigger(DestroyEvent, receiver, nil)
	return nil
}

func (receiver *Explosion) Reset() error {
	receiver.Object.Reset()
	if receiver.Ttl > 0 && receiver.throttle == nil {
		receiver.throttle = newThrottle(receiver.Ttl, false)
	}
	receiver.throttle.Reset()
	SwitchSprite(receiver.sprite, receiver.sprite)
	return nil
}

func (receiver *Explosion) DeSpawn() error {
	receiver.Trigger(DeSpawnEvent, receiver, nil)
	receiver.Object.DeSpawn()
	return nil
}

func (receiver *Explosion) GetDamage(target Vulnerable) (value int, owner ObjectInterface) {
	return receiver.Damage, nil
}

func (receiver *Explosion) Copy() *Explosion {
	instance := *receiver

	instance.ObservableObject 		= receiver.ObservableObject.Copy()
	instance.ObservableObject.Owner = instance
	instance.Object     	  		= receiver.Object.Copy()
	if instance.throttle != nil {
		instance.throttle		    = receiver.throttle.Copy()
	}

	return &instance
}

func NewExplosion2(obj *Object, oo *ObservableObject, Owner ObjectInterface) (*Explosion, error)  {
	instance := &Explosion{
		Object:     	  obj,
		ObservableObject: oo,
		Owner:			  Owner,
	}
	instance.ObservableObject.Owner = instance
	return instance, nil
}

