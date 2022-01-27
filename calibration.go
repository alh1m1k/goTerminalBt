package main

import (
	"GoConsoleBT/collider"
	"GoConsoleBT/controller"
	"math"
)

type Calibration struct {
	updater  *Updater
	render   Renderer
	collider *collider.Collider
	location *Location
	*ObservableObject
	*GameConfig
	internalChanel EventChanel
	closeChanel    bool
}

func (receiver *Calibration) Run(control *controller.Control) error {
	receiver.internalChanel = make(EventChanel)

	active, passive, _ := NewCalibrationBinary(control, receiver.internalChanel, 40, 20, 40, 40)

	receiver.collider.Remove(receiver.location)
	active.Reset()
	passive.Reset()
	receiver.spawn(active)
	receiver.spawn(passive)

	<-receiver.internalChanel

	active.Destroy(nil)
	passive.Destroy(nil)
	receiver.deSpawn(active)
	receiver.deSpawn(passive)
	receiver.collider.Add(receiver.location)

	err := receiver.process(active, passive)

	active.Free()
	passive.Free()
	close(receiver.internalChanel)
	if err == nil {
		receiver.Trigger(CalibrationCompleteEvent, receiver, nil)
	}
	return err
}

func (receiver *Calibration) spawn(object *CalibrationObject) {
	receiver.updater.Add(object)
	receiver.collider.Add(object)
	receiver.render.Add(object)
	object.Spawn()
}

func (receiver *Calibration) deSpawn(object *CalibrationObject) {
	receiver.updater.Remove(object)
	receiver.collider.Remove(object)
	receiver.render.Remove(object)
	object.DeSpawn()
}

func (receiver *Calibration) process(active, passive *CalibrationObject) error {

	colW := math.Abs(active.Probe[1].X+active.Probe[3].X) / 2
	rowH := math.Abs(active.Probe[0].Y+active.Probe[2].Y) / 2
	avgW := math.Abs(active.Probe[4].X-active.Probe[6].X) + colW
	avgH := math.Abs(active.Probe[4].Y-active.Probe[6].Y) + rowH

	if receiver.GameConfig == nil {
		receiver.GameConfig, _ = NewDefaultGameConfig()
	}

	receiver.GameConfig.ColWidth = colW
	receiver.GameConfig.RowHeight = rowH

	receiver.GameConfig.Box.X = 0
	receiver.GameConfig.Box.Y = 0
	receiver.GameConfig.Box.W = avgW
	receiver.GameConfig.Box.H = avgH

	return nil
}

func (receiver *Calibration) End() error {
	if receiver.closeChanel {
		close(receiver.GetEventChanel())
	}
	var err error
	if receiver.GameConfig != nil {
		_, err = saveConfig(receiver.GameConfig)
	}
	return err
}

func NewCalibration(updater *Updater, render Renderer, collider *collider.Collider, location *Location, chanel EventChanel) (*Calibration, error) {
	instance := &Calibration{
		updater:          updater,
		render:           render,
		collider:         collider,
		location:         location,
		ObservableObject: nil,
		GameConfig:       nil,
		internalChanel:   make(EventChanel),
		closeChanel:      false,
	}

	if chanel == nil {
		chanel = make(EventChanel)
		instance.closeChanel = true
	}

	instance.ObservableObject, _ = NewObservableObject(chanel, instance)

	return instance, nil
}
