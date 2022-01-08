package main

import (
	"GoConsoleBT/collider"
	"github.com/tanema/ump"
	"math/rand"
	"time"
)

type Projectile struct {
	*MotionObject
	*State
	*ObservableObject
	*throttle
	collisionsCnt     int64
	Owner             ObjectInterface
	Damage, DotDamage int
	Ttl               time.Duration
}

func (receiver *Projectile) ApplyState(current *StateItem) error {
	state := current.StateInfo.(*UnitStateInfo)
	SwitchSprite(state.sprite, receiver.sprite)
	receiver.sprite = state.sprite
	receiver.GetClBody().Resize(state.collisionW, state.collisionH)
	//for now resize only work if body not spawn
	return nil
}

func (receiver *Projectile) Update(timeLeft time.Duration) error {
	if receiver.destroyed {
		return nil
	}

	receiver.MotionObject.Update(timeLeft)

	if receiver.throttle != nil && receiver.throttle.Reach(timeLeft) {
		receiver.Destroy(nil)
	}

	return nil
}

func (receiver *Projectile) ApplySpeed(Speed Point) error {
	receiver.Speed = receiver.Speed.Plus(Speed)
	receiver.MaxSpeed = receiver.MaxSpeed.Plus(Speed)
	receiver.MinSpeed = receiver.MinSpeed.Plus(Speed)
	return nil
}

func (receiver *Projectile) OnTickCollide(object collider.Collideable, collision *ump.Collision, owner *collider.Interactions) {
	if receiver.DotDamage > 0 {
		//todo refactor this
		if receiver.HasTag("danger") && object.HasTag("vulnerable") {
			if object.HasTag("player") && DEBUG_IMMORTAL_PLAYER {
				return
			}
			DotDamage.Tag = receiver.tag
			DotDamage.Damage = receiver.DotDamage
			DotDamage.From = receiver
			object.(Vulnerable).ReciveDamage(&DotDamage)
		}
	}
}

func (receiver *Projectile) OnStartCollide(object collider.Collideable, collision *ump.Collision, owner *collider.Interactions) {
	if object.HasTag("obstacle") && !object.HasTag("low") {
		if !object.HasTag(receiver.GetAttr().TeamTag) {
			if !receiver.HasTag("projectile-penetrate") {
				receiver.Destroy(nil)
			} else {
				//todo penetrateCnt logic
			}
		}
	}
	if object.HasTag("border") {
		receiver.Destroy(nil)
	}
	receiver.collisionsCnt++
}

func (receiver *Projectile) OnStopCollide(object collider.Collideable, duration time.Duration, owner *collider.Interactions) {

}

func (receiver *Projectile) GetDamage(target Vulnerable) (value int, owner ObjectInterface) {
	if receiver.destroyed || target.HasTag(receiver.GetAttr().TeamTag) {
		return 0, receiver.Owner
	} else {
		return receiver.Damage, receiver.Owner
	}
}

func (receiver *Projectile) Destroy(nemesis ObjectInterface) error {
	if receiver.destroyed {
		return nil
	}
	receiver.MotionObject.Destroy(nemesis)
	receiver.Trigger(DestroyEvent, receiver, nil)
	return nil
}

func (receiver *Projectile) Reset() error {
	receiver.MotionObject.Reset()

	//bypass speed bug caused by applySpeed, restore from prototype
	prototype := receiver.Prototype.(*Projectile)
	receiver.MotionObject.Speed = prototype.Speed
	receiver.MotionObject.MaxSpeed = prototype.MaxSpeed
	receiver.MotionObject.MinSpeed = prototype.MinSpeed

	//warn bug zero life time if mix with receiver.throttle != nil
	if receiver.Ttl > 0 {
		receiver.throttle = newThrottle(receiver.Ttl, false)
	} else {
		logger.Printf("infinity projectile warning")
	}
	if receiver.throttle != nil {
		receiver.throttle.Reset()
	} else {

	}
	receiver.moving = true

	//todo guidance nav
	//this because of collision resize
	if receiver.Moving.Direction.X > 0 {
		receiver.Enter("right")
	}
	if receiver.Moving.Direction.X < 0 {
		receiver.Enter("left")
	}
	if receiver.Moving.Direction.Y < 0 {
		receiver.Enter("top")
	}
	if receiver.Moving.Direction.Y > 0 {
		receiver.Enter("bottom")
	}

	return nil
}

func (receiver *Projectile) DeSpawn() error {
	receiver.Trigger(DeSpawnEvent, receiver, nil)
	receiver.MotionObject.DeSpawn()
	return nil
}

func (receiver *Projectile) Spawn() error {
	receiver.MotionObject.Spawn()
	receiver.moving = true

	return nil
}

func (receiver *Projectile) GetOwner() ObjectInterface {
	return receiver.Owner
}

func (receiver *Projectile) Copy() *Projectile {
	instance := *receiver

	instance.ObservableObject = receiver.ObservableObject.Copy()
	instance.ObservableObject.Owner = &instance
	instance.MotionObject = receiver.MotionObject.Copy()
	instance.State = receiver.State.Copy()
	instance.State.Owner = &instance
	instance.Interactions.Subscribe(&instance)
	if instance.throttle != nil {
		instance.throttle = receiver.throttle.Copy()
	}

	return &instance
}

func NewProjectile2(mo *MotionObject, oo *ObservableObject, state *State, Owner ObjectInterface) (*Projectile, error) {
	instance := &Projectile{
		MotionObject:     mo,
		State:            state,
		ObservableObject: oo,
		Owner:            Owner,
	}
	instance.Interactions.Subscribe(instance)
	instance.State.Owner = instance
	instance.ObservableObject.Owner = instance
	instance.moving = true
	instance.destroyed = false
	instance.spawned = false
	instance.collisionsCnt = 0
	return instance, nil
}

var conventionalProjectileNames []string = []string{
	"tank-base-projectile-he",
	"tank-base-projectile-fanout",
	"tank-base-projectile-rail",
	"tank-base-projectile-flak",
}

func GetConventionalProjectileName() string {
	return conventionalProjectileNames[rand.Intn(len(conventionalProjectileNames))]
}
