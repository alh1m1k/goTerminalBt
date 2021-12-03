package main

import (
	"time"
)

type Updateable interface {
	Update(timeLeft time.Duration) error
}

type Updater struct {
	queue []Updateable
}

func (receiver *Updater) Add(object Updateable) {
	receiver.queue = append(receiver.queue, object)
}

func (receiver *Updater) Remove(object Updateable) {
	for indx, candidate := range receiver.queue {
		if object == candidate {
			receiver.queue[indx] = nil
		}
	}
}

func (receiver *Updater) Execute(timeLeft time.Duration) {
	for _, object := range receiver.queue {
		if object == nil {
			continue
		}
		object.Update(timeLeft)
	}
}

func NewUpdater(queueSize int) (*Updater, error) {
	return &Updater{
		queue: make([]Updateable, queueSize, queueSize),
	}, nil
}
