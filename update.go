package main

import (
	"time"
)

type Updater struct {
	queue []ObjectInterface
}

func (receiver *Updater) Add(object ObjectInterface) {
	receiver.queue = append(receiver.queue, object)
}

func (receiver *Updater) Remove(object ObjectInterface) {
	for indx, candidate := range receiver.queue {
		if object == candidate {
			//logger.Println("remove from update", object)
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
		queue: make([]ObjectInterface, queueSize, queueSize),
	}, nil
}
