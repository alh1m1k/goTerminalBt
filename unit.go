package main

import (
	"GoConsoleBT/collider"
	"GoConsoleBT/controller"
	"errors"
	"github.com/tanema/ump"
	"math"
	"time"
)

const UNIT_EVENT_FIRE = 100
const UNIT_EVENT_DAMAGE = 101
const UNIT_EVENT_ONSIGTH = 102
const UNIT_EVENT_OFFSIGTH = 103

var FireEvent Event = Event{
	EType:   UNIT_EVENT_FIRE,
	Payload: nil,
}

var DamadgeEvent Event = Event{
	EType:   UNIT_EVENT_DAMAGE,
	Payload: nil,
}

var OnSightEvent Event = Event{
	EType:   UNIT_EVENT_ONSIGTH,
	Payload: nil,
}

var OffSightEvent Event = Event{
	EType:   UNIT_EVENT_OFFSIGTH,
	Payload: nil,
}

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

	if command.CType == controller.CTYPE_DIRECTION || command.CType == controller.CTYPE_MOVE {
		receiver.moving = command.Action
	}

	if command.CType == controller.CTYPE_DIRECTION {
		if command.Pos.X != command.Pos.Y {
			receiver.Moving.Direction.X = command.Pos.X
			receiver.Moving.Direction.Y = command.Pos.Y
			receiver.Moving.Direction.X = math.Max(math.Min(receiver.Moving.Direction.X, 1), -1)
			receiver.Moving.Direction.Y = math.Max(math.Min(receiver.Moving.Direction.Y, 1), -1)
		} else {
			//invalid direction
		}
	}

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

	if command.CType == controller.CTYPE_FIRE && command.Action {
		err := receiver.Gun.Fire()
		if errors.Is(err, OutAmmoError) {
			receiver.Gun.Downgrade()
		}
		if errors.Is(err, GunConfigError) {
			logger.Println(err)
		}
	}

	if command.CType == controller.CTYPE_SPEED_FACTOR {
		receiver.speedAccelerator = Point(command.Pos) //x y -> are they same
	}

	return nil
}

func (receiver *Unit) Update(timeLeft time.Duration) error {
	if DEBUG_FREEZ_AI && receiver.HasTag("ai") {
		return nil
	}
	receiver.MotionObject.Update(timeLeft)
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
			logger.Print("seen")
		}
	} else {
		if object.HasTag("danger") && receiver.HasTag("vulnerable") {
			if DEBUG_IMMORTAL_PLAYER && receiver.HasTag("player") {
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
			logger.Print("unseen")
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
		receiver.Trigger(DamadgeEvent, receiver, damage)
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
	receiver.State.Reset()
	receiver.Gun.Reset()
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

func (receiver *Unit) Free() {
	receiver.ControlledObject.Free()
	receiver.MotionObject.Free()
	receiver.State.Free()
}

func (receiver *Unit) Copy() *Unit {
	instance := *receiver
	var control controller.Controller

	if DEBUG_NO_AI {
		control, _ = controller.NewNoneControl()
	} else {
		control, _ = AIBUILDER()
		control.(*BehaviorControl).AttachTo(&instance)
	}
	instance.ControlledObject, _ = NewControlledObject(control, &instance)

	instance.ObservableObject = receiver.ObservableObject.Copy()
	instance.ObservableObject.Owner = &instance
	instance.MotionObject = receiver.MotionObject.Copy()
	instance.State = receiver.State.Copy()
	instance.State.Owner = &instance
	instance.Gun = receiver.Gun.Copy()
	instance.Gun.Owner = &instance
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

func GetUnitState(id string) (*State, error) {
	return GetState(id, func(m map[string]interface{}) (interface{}, error) {
		var sprite Spriteer = nil
		var err error

		if animation, ok := m["animation"]; ok {
			//todo refactor this shit
			animationInfo := animation.(map[string]interface{})
			sprite, _ = GetAnimation(animationInfo["name"].(string), int(animationInfo["length"].(float64)), true, false)
			if sprite != nil {
				spriteAsAnimation := sprite.(*Animation)
				spriteAsAnimation.Cycled = animationInfo["cycled"].(bool)
				spriteAsAnimation.Duration = time.Duration(animationInfo["duration"].(float64))
				if blink, ok := animationInfo["blink"]; ok {
					spriteAsAnimation.BlinkRate = time.Duration(blink.(float64))
				}
			}
		}
		if sprite == nil {
			sprite, err = GetSprite(m["sprite"].(string), true, false)
			if err != nil {
				return nil, err
			}
		}

		return &UnitStateInfo{
			sprite:     sprite,
			collisionX: 0,
			collisionY: 0,
			collisionW: 0,
			collisionH: 0,
		}, nil
	})
}
