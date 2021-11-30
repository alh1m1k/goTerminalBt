package main

import (
	"GoConsoleBT/controller"
	"errors"
	"math"
	"time"
)

const TANK_EVENT_FIRE = 100
const TANK_EVENT_DAMADGE = 101

var FireEvent Event = Event{
	EType:   TANK_EVENT_FIRE,
	Payload: nil,
}

var DamadgeEvent Event = Event{
	EType:   TANK_EVENT_DAMADGE,
	Payload: nil,
}

type UnitStateInfo struct {
	sprite    Spriteer
	collisionX, collisionY, collisionW, collisionH float64
}

type UnitConfig struct {
	MotionObjectConfig
	State      *State
	Output     EventChanel
	Controller *controller.Control
	HP, FullHP, Score   int
}

type Unit struct {
	*ControlledObject
	*ObservableObject
	*MotionObject
	*State
	*Gun
	HP, FullHP, Score 	int
	projectile 			string
}

func (receiver *Unit) Execute(command controller.Command) error  {

	receiver.moving	= command.Move

	if command.Direction.X != command.Direction.Y {
		receiver.Move.Direction.X = command.Direction.X
		receiver.Move.Direction.Y = command.Direction.Y
		receiver.Move.Direction.X = math.Max(math.Min(receiver.Move.Direction.X, 1), -1)
		receiver.Move.Direction.Y = math.Max(math.Min(receiver.Move.Direction.Y, 1), -1)
	} else {
		//invalid direction
	}

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

	if command.Fire {
		err := receiver.Gun.Fire()
		if errors.Is(err, OutAmmoError) {
			receiver.Gun.Downgrade()
		}
		if errors.Is(err, GunConfigError) {
			logger.Println(err)
		}
		//receiver.Trigger(FireEvent, receiver, nil)
	}

	return nil
}

func (receiver *Unit) Update(timeLeft time.Duration) error {
	if receiver.destroyed {
		return nil
	}
	receiver.MotionObject.Update(timeLeft)
	if receiver.collision.Collided() {
		for object, _ := range receiver.collision.CollisionInfo().I() {
			if object.HasTag("danger") {
				if DEBUG_IMMORTAL_PLAYER && receiver.HasTag("player") {
					continue
				}
				receiver.ReciveDamage(object.(Danger))
			}
		}
	}
	return nil
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
	receiver.ControlledObject.deactivate()
	receiver.MotionObject.Destroy(nemesis)
	receiver.Trigger(DestroyEvent, receiver, nemesis)
	return nil
}

func (receiver *Unit) Reset() error {
	receiver.ControlledObject.activate()
	receiver.MotionObject.Reset()
	receiver.State.Reset()
	receiver.Gun.Reset()
	receiver.HP = receiver.FullHP
	receiver.moving = false
	receiver.Trigger(ResetEvent, receiver, nil)
	return nil
}

func (receiver *Unit) DeSpawn() error {
	receiver.MotionObject.DeSpawn()
	receiver.Trigger(DeSpawnEvent, receiver, nil)
	return nil
}

func (receiver *Unit) Spawn() error {
	receiver.MotionObject.Spawn()
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

func (receiver *Unit) Free()  {
	receiver.ControlledObject.Free()
	receiver.MotionObject.Free()
	receiver.State.Free()
}

func (receiver *Unit) Copy() *Unit {
	instance := *receiver
	var control *controller.Control

	if DEBUG_NO_AI {
		control, _ = controller.NewNoneControl()
	} else {
		control, _ = controller.NewAIControl()
	}
	instance.ControlledObject, _ = NewControlledObject(control, &instance)

	instance.ObservableObject = receiver.ObservableObject.Copy()
	instance.ObservableObject.Owner = &instance
	instance.MotionObject     = receiver.MotionObject.Copy()
	instance.State 			  = receiver.State.Copy()
	instance.State.Owner	  = &instance
	instance.Gun			  = receiver.Gun.Copy()
	instance.Gun.Owner 		  = &instance

	return &instance
}

func (receiver *Unit) GetEventChanel() EventChanel  {
	return receiver.output
}

func NewUnit2(co *ControlledObject, oo *ObservableObject,
			  mo *MotionObject, st *State) (*Unit, error)  {

	gun, _ := NewGun(nil)
	instance := &Unit{
		ControlledObject: co,
		ObservableObject: oo,
		MotionObject:     mo,
		State:			  st,
		Gun:			  gun,
	}

	instance.Gun.Owner					= instance
	if st != nil {
		instance.State.Owner 			= instance
	}
	if co != nil {
		instance.ControlledObject.Owner = instance
	}
	if oo != nil {
		instance.ObservableObject.Owner = instance
	}

	return instance, nil
}

func GetTankState(id string) (*State,error)  {
	return GetState(id, func(m map[string]interface{}) (interface{}, error) {
		var sprite Spriteer = nil
		var err error

		if animation, ok := m["animation"]; ok {
			//todo refactor this shit
			animationInfo 	:= animation.(map[string]interface{})
			sprite, _ 		= GetAnimation(animationInfo["name"].(string), int(animationInfo["length"].(float64)), true, false)
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
