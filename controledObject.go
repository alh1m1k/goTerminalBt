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
	*controller.Control
	terminator chan bool
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

func (receiver *ControlledObject) deactivate() error {
	if receiver.dispatcherEnable {
		close(receiver.terminator)
	}
	receiver.dispatcherEnable = false
	return nil
}

func (receiver *ControlledObject) activate() error {
	if !receiver.dispatcherEnable {
		if receiver.Control == nil {
			logger.Println("command chanel not found")
			return CommandChanelNotFoundError
		}
		receiver.terminator = make(chan bool)
		go coCmdDispatcher(receiver, receiver.GetCommandChanel(), receiver.terminator)
	}
	receiver.dispatcherEnable = true
	return nil
}

func (receiver *ControlledObject) Free() error {
	close(receiver.terminator)
	return nil
}

func (receiver *ControlledObject) Copy() *ControlledObject {
	instance := *receiver
	return &instance
}

func NewControlledObject(cmd *controller.Control, owner ControlledObjectInterface) (*ControlledObject, error) {
	instance := new(ControlledObject)

	instance.Owner = owner
	instance.Control = cmd

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
