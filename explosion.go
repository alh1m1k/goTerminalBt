package main

import (
	"GoConsoleBT/collider"
	"github.com/alh1m1k/ump"
	"time"
)

type DamageProxy struct {
	*Tags
	From   ObjectInterface
	Damage int
}

func (receiver *DamageProxy) GetDamage(target Vulnerable) (value int, nemesis ObjectInterface) {
	return receiver.Damage, receiver.From
}

var (
	DotDamage = DamageProxy{
		Tags:   nil,
		From:   nil,
		Damage: 0,
	}
	NoDamage = MinMax{0, 0}
)

type Explosion struct {
	*Object
	*ObservableObject
	*throttle
	RangeDamageReductionFunction, RangeDotDamageReductionFunction timeFunction
	Owner                                                         ObjectInterface
	Damage, DotDamage                                             int
	Ttl                                                           time.Duration
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

func (receiver *Explosion) OnTickCollide(object collider.Collideable, collision *ump.Collision, owner *collider.Interactions) {
	if receiver.DotDamage > 0 {
		//todo refactor this
		if receiver.HasTag("danger") && object.HasTag("vulnerable") {
			if object.HasTag("player") && DEBUG_IMMORTAL_PLAYER {
				return
			}
			DotDamage.Tags = receiver.Tags
			DotDamage.Damage = receiver.DotDamage
			DotDamage.From = receiver
			object.(Vulnerable).ReciveDamage(&DotDamage)
		}
	}
}

func (receiver *Explosion) OnStartCollide(object collider.Collideable, collision *ump.Collision, owner *collider.Interactions) {
	if receiver.HasTag("danger") && object.HasTag("vulnerable") {
		if DEBUG_IMMORTAL_PLAYER && (object.HasTag("player") || object.HasTag("base")) {

		} else {
			object.(Vulnerable).ReciveDamage(receiver)
		}
	}
}

func (receiver *Explosion) OnStopCollide(object collider.Collideable, duration time.Duration, owner *collider.Interactions) {

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
	instance.ObservableObject.Owner = &instance
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
