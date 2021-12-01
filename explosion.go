package main

import (
	"GoConsoleBT/collider"
	"github.com/tanema/ump"
	"time"
)

type DotDamageProxy struct {
	Tag    []string
	From   ObjectInterface
	Damage int
}

func (receiver *DotDamageProxy) GetDamage(target Vulnerable) (value int, nemesis ObjectInterface) {
	return receiver.Damage, receiver.From
}

func (receiver *DotDamageProxy) HasTag(tag string) bool {
	for _, part := range receiver.Tag {
		if part == tag {
			return true
		}
	}
	return false
}

var DotDamage = DotDamageProxy{
	Tag:    nil,
	From:   nil,
	Damage: 0,
}

type Explosion struct {
	*Object
	*ObservableObject
	*throttle
	Owner     ObjectInterface
	Damage    int
	DotDamage int
	Ttl       time.Duration
}

func (receiver *Explosion) ApplyState(current *StateItem) error {
	receiver.sprite = current.StateInfo.(*UnitStateInfo).sprite
	return nil
}

func (receiver *Explosion) Update(timeLeft time.Duration) error {
	if receiver.destroyed {
		return nil
	}
	receiver.Object.Update(timeLeft)

	if receiver.throttle != nil && receiver.throttle.Reach(timeLeft) {
		receiver.Destroy(nil)
	}

	return nil
}

func (receiver *Explosion) OnTickCollide(object collider.Collideable, collision *ump.Collision) {

	if receiver.DotDamage > 0 {
		//todo refactor this
		if object.HasTag("vulnerable") {
			DotDamage.Tag = receiver.tag
			DotDamage.Damage = receiver.DotDamage
			DotDamage.From = receiver
			object.(Vulnerable).ReciveDamage(&DotDamage)
		}
	}
}

func (receiver *Explosion) OnStartCollide(object collider.Collideable, collision *ump.Collision) {

}

func (receiver *Explosion) OnStopCollide(object collider.Collideable, duration time.Duration) {

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

	instance.ObservableObject = receiver.ObservableObject.Copy()
	instance.ObservableObject.Owner = instance
	instance.Object = receiver.Object.Copy()
	if instance.throttle != nil {
		instance.throttle = receiver.throttle.Copy()
	}
	instance.Interactions.Subscribe(&instance)
	return &instance
}

func NewExplosion2(obj *Object, oo *ObservableObject, Owner ObjectInterface) (*Explosion, error) {
	instance := &Explosion{
		Object:           obj,
		ObservableObject: oo,
		Owner:            Owner,
	}
	instance.ObservableObject.Owner = instance
	instance.Interactions.Subscribe(instance)
	return instance, nil
}
