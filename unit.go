package main

import (
	"GoConsoleBT/collider"
	"GoConsoleBT/controller"
	"errors"
	"github.com/alh1m1k/ump"
	"math"
	"time"
)

const UNIT_EVENT_FIRE = 100
const UNIT_EVENT_DAMAGE = 101
const UNIT_EVENT_ONSIGTH = 102
const UNIT_EVENT_OFFSIGTH = 103

var (
	FireEvent Event = Event{
		EType:   UNIT_EVENT_FIRE,
		Payload: nil,
	}
	DamageEvent Event = Event{
		EType:   UNIT_EVENT_DAMAGE,
		Payload: nil,
	}
	OnSightEvent Event = Event{
		EType:   UNIT_EVENT_ONSIGTH,
		Payload: nil,
	}
	OffSightEvent Event = Event{
		EType:   UNIT_EVENT_OFFSIGTH,
		Payload: nil,
	}
	InvalidDirectionError = errors.New("invalid direction")
)

type UnitStateInfo struct {
	sprite                                         Spriteer
	collisionX, collisionY, collisionW, collisionH float64
}

type Unit struct {
	*ControlledObject
	*ObservableObject
	*MotionObject
	*State
	*Gun
	vision             *collider.ClBody
	VisionInteractions *collider.Interactions
	HP, FullHP, Score  int
	projectile         string
}

func (receiver *Unit) Execute(command controller.Command) error {

	if command.CType == controller.CTYPE_MOVE {
		receiver.moving = command.Action
	}

	if command.CType == controller.CTYPE_DIRECTION || command.CType == controller.CTYPE_MOVE {
		if command.Pos != controller.PosIrrelevant {
			receiver.AlignToDirection(Point(command.Pos))
		}
	}

	if command.CType == controller.CTYPE_FIRE && command.Action {
		if receiver.Gun != nil {
			err := receiver.Gun.Fire()
			if errors.Is(err, OutAmmoError) {
				receiver.Gun.Downgrade()
			}
			if errors.Is(err, GunConfigError) {
				logger.Println(err)
			}
		} else {
			logger.Println("fire command receive but no gun to fire")
		}
	}

	if command.CType == controller.CTYPE_SPEED_FACTOR {
		receiver.speedAccelerator = Point(command.Pos) //x y -> are they same
	}

	return nil
}

func (receiver *Unit) Update(timeLeft time.Duration) error {
	if DEBUG_FREEZ_AI && receiver.HasTag("ai") {
		receiver.Object.Update(timeLeft)
	} else {
		receiver.MotionObject.Update(timeLeft)
	}
	if receiver.vision != nil && receiver.VisionInteractions != nil {
		ccx, ccy := receiver.collision.GetCenter()
		cvx, cvy := receiver.vision.GetCenter()
		receiver.vision.RelativeMove(ccx-cvx, ccy-cvy) //pos correction after collide
		receiver.VisionInteractions.Interact(receiver.vision, timeLeft)
	}
	return nil
}

func (receiver *Unit) OnTickCollide(object collider.Collideable, collision *ump.Collision, owner *collider.Interactions) {

}

func (receiver *Unit) OnStartCollide(object collider.Collideable, collision *ump.Collision, owner *collider.Interactions) {
	if owner == receiver.VisionInteractions {
		//todo change tag tank
		if object.HasTag("tank") && !object.HasTag(receiver.GetAttr().TeamTag) {
			receiver.Trigger(OnSightEvent, receiver, object)
		}
	} else {
		if object.HasTag("danger") && receiver.HasTag("vulnerable") {
			if DEBUG_IMMORTAL_PLAYER && (receiver.HasTag("player") || receiver.HasTag("base")) {
				return
			}
			receiver.ReciveDamage(object.(Danger))
		}
	}
}

func (receiver *Unit) OnStopCollide(object collider.Collideable, duration time.Duration, owner *collider.Interactions) {
	if owner == receiver.VisionInteractions {
		if object.HasTag("tank") && !object.HasTag(receiver.GetAttr().TeamTag) {
			receiver.Trigger(OffSightEvent, receiver, object)
		}
	} else {

	}
}

func (receiver *Unit) GetVision() *collider.ClBody {
	return receiver.vision
}

func (receiver *Unit) Move(x, y float64) {
	receiver.MotionObject.Move(x, y)
	if receiver.vision != nil {
		ccx, ccy := receiver.collision.GetCenter()
		cvx, cvy := receiver.vision.GetCenter()
		receiver.vision.RelativeMove(ccx-cvx, ccy-cvy)
	}
}

func (receiver *Unit) RelativeMove(x, y float64) {
	receiver.MotionObject.RelativeMove(x, y)
	if receiver.vision != nil {
		ccx, ccy := receiver.collision.GetCenter()
		cvx, cvy := receiver.vision.GetCenter()
		receiver.vision.RelativeMove(ccx-cvx, ccy-cvy)
	}
}

func (receiver *Unit) ReciveDamage(incoming Danger) {
	damage, nemesis := incoming.GetDamage(receiver)
	if damage <= 0 {
		return
	}
	receiver.HP -= damage
	if receiver.HP <= 0 {
		receiver.Destroy(nemesis)
	} else {
		receiver.Trigger(DamageEvent, receiver, damage)
	}
}

func (receiver *Unit) GetScore() int {
	return receiver.Score
}

func (receiver *Unit) Destroy(nemesis ObjectInterface) error {
	if receiver.destroyed {
		return nil
	} //collision in cycle may cause multiple destroy
	receiver.ControlledObject.Deactivate()
	receiver.MotionObject.Destroy(nemesis)
	receiver.Trigger(DestroyEvent, receiver, nemesis)
	return nil
}

func (receiver *Unit) Reset() error {
	receiver.MotionObject.Reset()
	if receiver.State != nil {
		receiver.State.Reset()
		receiver.AlignToDirection(receiver.Direction)
	}
	if receiver.Gun != nil {
		receiver.Gun.Reset()
	}
	receiver.HP = receiver.FullHP
	receiver.moving = false
	receiver.Trigger(ResetEvent, receiver, nil)
	return nil
}

func (receiver *Unit) DeSpawn() error {
	if receiver.Control != nil {
		receiver.ControlledObject.Deactivate()
	}
	receiver.MotionObject.DeSpawn()
	receiver.Trigger(DeSpawnEvent, receiver, nil)
	return nil
}

func (receiver *Unit) Spawn() error {
	receiver.MotionObject.Spawn()
	receiver.ControlledObject.Activate()
	receiver.Trigger(SpawnEvent, receiver, nil)
	return nil
}

func (receiver *Unit) GetAppearDuration() time.Duration {
	return 8 * time.Second
}

func (receiver *Unit) ApplyState(current *StateItem) error {
	SwitchSprite(current.StateInfo.(*UnitStateInfo).sprite, receiver.sprite)
	receiver.sprite = current.StateInfo.(*UnitStateInfo).sprite
	return nil
}

func (receiver *Unit) AlignToDirection(Pos Point) error {
	/*	if Pos == receiver.Moving.Direction {
		return nil
	}*/
	if Pos.X != Pos.Y {
		receiver.Moving.Direction.X = math.Max(math.Min(Pos.X, 1), -1)
		receiver.Moving.Direction.Y = math.Max(math.Min(Pos.Y, 1), -1)
	} else {
		return InvalidDirectionError
	}
	receiver.Moving.Direction = Pos
	if receiver.State != nil {
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
	}
	return nil
}

func (receiver *Unit) Free() {
	receiver.ControlledObject.Free()
	receiver.MotionObject.Free()
	if receiver.State != nil {
		receiver.State.Free()
	}
}

func (receiver *Unit) Copy() *Unit {
	instance := *receiver

	if receiver.ControlledObject != nil {
		instance.ControlledObject = instance.ControlledObject.Copy()
		instance.ControlledObject.Owner = &instance
	}
	if instance.State != nil {
		instance.State = receiver.State.Copy()
		instance.State.Owner = &instance
	}
	instance.ObservableObject = receiver.ObservableObject.Copy()
	instance.ObservableObject.Owner = &instance
	instance.MotionObject = receiver.MotionObject.Copy()
	if receiver.Gun != nil {
		instance.Gun = receiver.Gun.Copy()
		instance.Gun.Owner = &instance
	}
	instance.Interactions.Subscribe(&instance)

	if receiver.vision != nil {
		instance.vision = receiver.vision.Copy()
	}
	if receiver.VisionInteractions != nil {
		instance.VisionInteractions = receiver.VisionInteractions.Copy()
		instance.VisionInteractions.Subscribe(&instance)
	}

	return &instance
}

func (receiver *Unit) GetEventChanel() EventChanel {
	return receiver.output
}

func NewUnit(co *ControlledObject, oo *ObservableObject,
	mo *MotionObject, st *State, vision *collider.ClBody) (*Unit, error) {

	gun, _ := NewGun(nil)
	instance := &Unit{
		ControlledObject: co,
		ObservableObject: oo,
		MotionObject:     mo,
		State:            st,
		Gun:              gun,
		vision:           vision,
	}

	instance.Interactions.Subscribe(instance)
	if vision != nil {
		instance.VisionInteractions, _ = collider.NewIteractions()
		instance.VisionInteractions.Subscribe(instance)
	}

	instance.Gun.Owner = instance
	if st != nil {
		instance.State.Owner = instance
	}
	if co != nil {
		instance.ControlledObject.Owner = instance
	}
	if oo != nil {
		instance.ObservableObject.Owner = instance
	}

	return instance, nil
}
