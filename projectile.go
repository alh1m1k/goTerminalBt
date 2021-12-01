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
	collisionsCnt int64
	Owner         ObjectInterface
	Damage        int
	Ttl           time.Duration
}

func (receiver *Projectile) ApplyState(current *StateItem) error {
	receiver.sprite = current.StateInfo.(*UnitStateInfo).sprite
	return nil
}

func (receiver *Projectile) Update(timeLeft time.Duration) error {
	if receiver.destroyed {
		return nil
	}
	collision := receiver.collision

	receiver.Interactions.Interact(receiver, timeLeft)

	if receiver.throttle != nil && receiver.throttle.Reach(timeLeft) {
		receiver.Destroy(nil)
	}

	if receiver.moving {
		collision.RelativeMove(
			receiver.Move.Direction.X*receiver.Move.Speed.X/float64(TIME_FACTOR),
			receiver.Move.Direction.Y*receiver.Move.Speed.Y/float64(TIME_FACTOR),
		)
		if receiver.AccelDuration > 0 {
			fraction := receiver.AccelTimeFunc(float64(receiver.currAccelDuration) / float64(receiver.AccelDuration))
			receiver.Move.Speed.X = receiver.MinSpeed.X + ((receiver.MaxSpeed.X - receiver.MinSpeed.X) * fraction)
			receiver.Move.Speed.Y = receiver.MinSpeed.Y + ((receiver.MaxSpeed.Y - receiver.MinSpeed.Y) * fraction)
			receiver.currAccelDuration += timeLeft
			if receiver.currAccelDuration > receiver.AccelDuration {
				receiver.currAccelDuration = receiver.AccelDuration
			}
		}
	} else {
		receiver.currAccelDuration = 0
	}

	return nil
}

func (receiver *Projectile) GetZIndex() int {
	return 0
}

func (receiver *Projectile) OnTickCollide(object collider.Collideable, collision *ump.Collision) {

}

func (receiver *Projectile) OnStartCollide(object collider.Collideable, collision *ump.Collision) {
	if object.HasTag("obstacle") {
		if !object.HasTag(receiver.GetAttr().TeamTag) {
			if !receiver.HasTag("projectile-penetrate") {
				receiver.Destroy(nil)
			} else {
				//todo penetrateCnt logic
			}
		}
	}
	receiver.collisionsCnt++
}

func (receiver *Projectile) OnStopCollide(object collider.Collideable, duration time.Duration) {

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
	if receiver.Ttl > 0 {
		receiver.throttle = newThrottle(receiver.Ttl, false)
	} else {
		logger.Printf("infinity projectile warning")
	}
	if receiver.throttle != nil {
		receiver.throttle.Reset()
	}
	SwitchSprite(receiver.sprite, receiver.sprite)
	receiver.moving = true
	return nil
}

func (receiver *Projectile) DeSpawn() error {
	receiver.Trigger(DeSpawnEvent, receiver, nil)
	receiver.Object.DeSpawn()
	return nil
}

func (receiver *Projectile) Spawn() error {
	receiver.Object.Spawn()
	receiver.moving = true

	//todo guidance nav
	if receiver.Move.Direction.X > 0 {
		receiver.Enter("right")
	}
	if receiver.Move.Direction.X < 0 {
		receiver.Enter("left")
	}
	if receiver.Move.Direction.Y < 0 {
		receiver.Enter("top")
	}
	if receiver.Move.Direction.Y > 0 {
		receiver.Enter("bottom")
	}

	return nil
}

func (receiver *Projectile) GetOwner() ObjectInterface {
	return receiver.Owner
}

func (receiver *Projectile) Copy() *Projectile {
	instance := *receiver

	instance.ObservableObject = receiver.ObservableObject.Copy()
	instance.Owner = &instance
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
}

func GetConventionalProjectileName() string {
	return conventionalProjectileNames[rand.Intn(len(conventionalProjectileNames))]
}
