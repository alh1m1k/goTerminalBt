package main

import (
	"GoConsoleBT/collider"
	"GoConsoleBT/controller"
	"math"
	"time"
)

const CALIBRATION_COMPLETE = 600

var CalibrationCompleteEvent = Event{
	EType:   CALIBRATION_COMPLETE,
	Object:  nil,
	Payload: nil,
}

//receiver vs opposite
//bottom to top
//left to right
//top to bottom
//right to left

//borders
//left top
//right top
//right bottom
//left bottom

type CalibrationObject struct {
	*MotionObject
	*ControlledObject
	*ObservableObject
	opposite *CalibrationObject
	Probe    []*Point
}

func (receiver *CalibrationObject) Update(timeLeft time.Duration) error {
	if receiver.moving {
		receiver.GetClBody().RelativeMove(
			receiver.Moving.Direction.X*receiver.Moving.Speed.X*(float64(timeLeft)/float64(time.Second)),
			receiver.Moving.Direction.Y*receiver.Moving.Speed.Y*(float64(timeLeft)/float64(time.Second)),
		)
	}
	receiver.moving = false
	return nil
}

func (receiver *CalibrationObject) Execute(command controller.Command) error {

	if command.CType == controller.CTYPE_DIRECTION || command.CType == controller.CTYPE_MOVE {
		receiver.moving = command.Action
	}

	if command.Pos.X != command.Pos.Y {
		receiver.Moving.Direction.X = command.Pos.X
		receiver.Moving.Direction.Y = command.Pos.Y
		receiver.Moving.Direction.X = math.Max(math.Min(receiver.Moving.Direction.X, 1), -1)
		receiver.Moving.Direction.Y = math.Max(math.Min(receiver.Moving.Direction.Y, 1), -1)
	} else {
		//invalid direction
	}

	if command.CType == controller.CTYPE_FIRE && command.Action {
		receiver.moving = false
		receiver.Calibrate()
	}

	return nil
}

func (receiver *CalibrationObject) Calibrate() {
	index := len(receiver.Probe)

	x, y := receiver.GetClBody().GetXY()
	w, h := receiver.GetClBody().GetWH()
	if index < 4 {
		opX, opY := receiver.opposite.GetClBody().GetXY()
		receiver.Probe = append(receiver.Probe, &Point{
			X: math.Abs(x-opX) / w,
			Y: math.Abs(y-opY) / h,
		})
	} else if index < 8 {
		receiver.Probe = append(receiver.Probe, &Point{
			X: x,
			Y: y,
		})
	}

	if index == 7 {
		receiver.Trigger(CalibrationCompleteEvent, receiver, receiver.Probe)
	}
}

func (receiver *CalibrationObject) Reset() error {
	receiver.ControlledObject.Activate()
	receiver.MotionObject.Reset()
	receiver.moving = false
	return nil
}

func NewCalibrationObject(control *controller.Control, chanel EventChanel, x, y float64) (*CalibrationObject, error) {
	sprite := NewSprite()
	sprite.Write([]byte("****\n****\n****\n****"))
	collision := collider.NewPenetrateCollision(x, y, 4, 4)
	obj, _ := NewObject(sprite, collision)
	mm, _ := NewMotionObject(obj, Point{
		X: 0,
		Y: -1,
	}, Point{
		X: 8,
		Y: 8,
	})
	mm.MinSpeed.X = 8
	mm.MinSpeed.Y = 8
	mm.MaxSpeed.X = 16
	mm.MaxSpeed.Y = 16
	mm.AccelDuration = time.Second * 2
	co, _ := NewControlledObject(control, nil)
	oo, _ := NewObservableObject(chanel, nil)

	calibrationObject := &CalibrationObject{
		MotionObject:     mm,
		ControlledObject: co,
		ObservableObject: oo,
		opposite:         nil,
		Probe:            make([]*Point, 0, 8),
	}
	calibrationObject.ControlledObject.Owner = calibrationObject
	calibrationObject.ObservableObject.Owner = calibrationObject

	return calibrationObject, nil
}

func (receiver *CalibrationObject) Free() {
	receiver.ControlledObject.Free()
	receiver.MotionObject.Free()
}

func NewCalibrationBinary(control *controller.Control, chanel EventChanel, x1, y1, x2, y2 float64) (active *CalibrationObject, passive *CalibrationObject, error error) {
	none, _ := controller.NewNoneControl()
	obj1, _ := NewCalibrationObject(control, chanel, x1, y1)
	obj2, _ := NewCalibrationObject(none, nil, x2, y2)
	obj1.opposite = obj2
	obj2.opposite = obj1

	return obj1, obj2, nil
}
