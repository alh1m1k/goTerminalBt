package main

import (
	"GoConsoleBT/collider"
	"time"
)

type Move struct {
	Speed     Point
	Direction Point
}

type MotionObjectConfig struct {
	Position, Speed, Direction Point
	Sprite                     Spriteer
	Collision                  *collider.ClBody
	Team                       int8
}

type Motioner interface {
	GetSpeed() *Point
	GetDirection() *Point
}

type Accelerator interface {
	GetMaxSpeed() *Point
	GetMinSpeed() *Point
}

type MotionObject struct {
	*Object
	Move
	moving            bool
	AccelDuration     time.Duration
	AccelTimeFunc     timeFunction
	MinSpeed          Point
	MaxSpeed          Point
	currAccelDuration time.Duration
	accelerate        bool
}

func (receiver *MotionObject) Update(timeLeft time.Duration) error {
	if receiver.destroyed {
		return nil
	}
	receiver.Object.Update(timeLeft)

	if receiver.moving {
		receiver.collision.RelativeMove(
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

func (receiver *MotionObject) GetSpeed() *Point {
	return &receiver.Speed
}

func (receiver *MotionObject) GetMaxSpeed() *Point {
	return &receiver.MaxSpeed
}

func (receiver *MotionObject) GetMinSpeed() *Point {
	return &receiver.MinSpeed
}

func (receiver *MotionObject) GetDirection() *Point {
	return &receiver.Direction
}

func (receiver *MotionObject) Destroy(nemesis ObjectInterface) error {
	receiver.Object.Destroy(nemesis)
	receiver.moving = false
	return nil
}

func (receiver *MotionObject) Reset() error {
	receiver.Object.Reset()
	receiver.Speed = receiver.MinSpeed
	receiver.currAccelDuration = 0
	receiver.accelerate = true
	receiver.moving = false
	return nil
}

func (receiver *MotionObject) Copy() *MotionObject {
	instance := *receiver
	instance.Object = instance.Object.Copy()
	instance.Move = Move{
		Speed: Point{
			X: receiver.Move.Speed.X,
			Y: receiver.Move.Speed.Y,
		},
		Direction: Point{
			X: receiver.Move.Direction.X,
			Y: receiver.Move.Direction.Y,
		},
	}
	return &instance
}

func NewMotionObject(s Spriteer, c *collider.ClBody, direction Point, speed Point) (*MotionObject, error) {
	obj, _ := NewObject(s, c)
	instance := MotionObject{
		Object: obj,
		Move: Move{
			Speed:     speed,
			Direction: direction,
		},
		MinSpeed:          speed,
		MaxSpeed:          speed,
		AccelTimeFunc:     LinearTimeFunction,
		AccelDuration:     0,
		currAccelDuration: 0,
		accelerate:        true,
		moving:            false,
	}
	return &instance, nil
}

func NewMotionObject2(obj *Object, direction Point, speed Point) (*MotionObject, error) {
	instance := MotionObject{
		Object: obj,
		Move: Move{
			Speed:     speed,
			Direction: direction,
		},
		MinSpeed:          speed,
		MaxSpeed:          speed,
		AccelTimeFunc:     LinearTimeFunction,
		AccelDuration:     0,
		currAccelDuration: 0,
		accelerate:        true,
		moving:            false,
	}
	return &instance, nil
}
