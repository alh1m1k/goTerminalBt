package main

import (
	"time"
)

type Moving struct {
	Speed     Point
	Direction Point
}

type Motioner interface {
	GetSpeed() *Point
	GetDirection() *Point
}

type Accelerator interface {
	GetMaxSpeed() *Point
	GetMinSpeed() *Point
}

type MotionObjectInterface interface {
	ObjectInterface
	Motioner
	Accelerator
}

type MotionObject struct {
	*Object
	Moving
	speedAccelerator  Point //todo remove
	moving            bool
	AccelDuration     time.Duration
	AccelTimeFunc     timeFunction
	MinSpeed          Point
	MaxSpeed          Point
	currAccelDuration time.Duration
	alignToGrid       bool
	accelerate        bool
}

func (receiver *MotionObject) Update(timeLeft time.Duration) error {
	if receiver.destroyed {
		return nil
	}
	receiver.Object.Update(timeLeft)

	if receiver.moving {

		deltaX := receiver.Moving.Direction.X * (receiver.Moving.Speed.X * receiver.speedAccelerator.X) * (float64(timeLeft) / float64(time.Second))
		deltaY := receiver.Moving.Direction.Y * (receiver.Moving.Speed.Y * receiver.speedAccelerator.Y) * (float64(timeLeft) / float64(time.Second))

		receiver.RelativeMove(deltaX, deltaY)

		if receiver.AccelDuration > 0 {
			fraction := receiver.AccelTimeFunc(float64(receiver.currAccelDuration) / float64(receiver.AccelDuration))
			receiver.Moving.Speed.X = receiver.MinSpeed.X + ((receiver.MaxSpeed.X - receiver.MinSpeed.X) * fraction)
			receiver.Moving.Speed.Y = receiver.MinSpeed.Y + ((receiver.MaxSpeed.Y - receiver.MinSpeed.Y) * fraction)
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

func (receiver *MotionObject) Accelerate(factorX, factorY float64) {
	receiver.speedAccelerator.X, receiver.speedAccelerator.Y = factorX, factorY
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
	receiver.speedAccelerator.X, receiver.speedAccelerator.Y = 1.0, 1.0
	return nil
}

func (receiver *MotionObject) Copy() *MotionObject {
	instance := *receiver
	instance.Object = instance.Object.Copy()
	instance.Moving = Moving{
		Speed: Point{
			X: receiver.Moving.Speed.X,
			Y: receiver.Moving.Speed.Y,
		},
		Direction: Point{
			X: receiver.Moving.Direction.X,
			Y: receiver.Moving.Direction.Y,
		},
	}
	return &instance
}

func NewMotionObject(obj *Object, direction Point, speed Point) (*MotionObject, error) {
	instance := MotionObject{
		Object: obj,
		Moving: Moving{
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
		alignToGrid:       false,
		speedAccelerator:  Point{1.0, 1.0},
	}
	return &instance, nil
}
