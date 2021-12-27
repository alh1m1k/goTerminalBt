package main

import (
	"GoConsoleBT/controller"
	"errors"
)

var CommandChanelNotFoundError = errors.New("command chanel not found")

type ControlledObjectInterface interface {
	Execute(command controller.Command) error
}

type ControlledObject struct {
	Owner            ControlledObjectInterface
	dispatcherEnable bool
	Control          controller.Controller
	terminator       chan bool
}

func (receiver *ControlledObject) Execute(command controller.Command) error {
	if receiver.Owner != nil {
		if DEBUG_EXEC {
			logger.Printf("exec: %T, %+v \n", command, command)
		}
		return receiver.Owner.Execute(command)
	}
	return nil
}

func (receiver *ControlledObject) Deactivate() error {
	receiver.Control.Disable()
	/*	if bc, ok := receiver.Control.(*BehaviorControl); ok {
		bc.Deattach()
	}*/
	if receiver.dispatcherEnable {
		close(receiver.terminator)
	}
	receiver.dispatcherEnable = false
	return nil
}

func (receiver *ControlledObject) Activate() error {
	if !receiver.dispatcherEnable {
		if receiver.Control == nil {
			logger.Println("command chanel not found")
			return CommandChanelNotFoundError
		}
		receiver.terminator = make(chan bool)
		go coCmdDispatcher(receiver, receiver.Control.GetCommandChanel(), receiver.terminator)
	}
	receiver.dispatcherEnable = true
	if bc, ok := receiver.Control.(*BehaviorControl); ok {
		if unit, ok := receiver.Owner.(*Unit); ok {
			bc.AttachTo(unit) //todo simplify
		}
	}
	receiver.Control.Enable()
	return nil
}

func (receiver *ControlledObject) Free() error {
	close(receiver.terminator)
	return nil
}

func (receiver *ControlledObject) Copy() *ControlledObject {
	instance := *receiver
	instance.terminator = nil
	instance.dispatcherEnable = false
	if receiver.Control != nil {
		switch receiver.Control.(type) {
		case *controller.Control:
			instance.Control = receiver.Control.(*controller.Control).Copy()
		case *BehaviorControl:
			instance.Control = receiver.Control.(*BehaviorControl).Copy()
		default:
			logger.Println("unknown type of Control")
		}
	}
	if receiver.dispatcherEnable {
		logger.Println("dispatcher already enable")
		go coCmdDispatcher(receiver, receiver.Control.GetCommandChanel(), receiver.terminator)
	}
	return &instance
}

func NewControlledObject(cmd controller.Controller, owner ControlledObjectInterface) (*ControlledObject, error) {
	instance := new(ControlledObject)

	instance.Owner = owner
	instance.Control = cmd
	instance.dispatcherEnable = false

	return instance, nil
}

func coCmdDispatcher(object ControlledObjectInterface, cmdEvents <-chan controller.Command, termEvents chan bool) {
	if object == nil {
		return
	}
	for {
		select {
		case cmd, ok := <-cmdEvents:
			if !ok {
				return
			}
			if DEBUG_EVENT {
				logger.Printf("receive: %T, %+v \n", cmd, cmd)
			}
			object.Execute(cmd)
		case <-termEvents:
			return
		}
	}
}
