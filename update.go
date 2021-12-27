package main

import (
	"time"
)

type Updateable interface {
	Update(timeLeft time.Duration) error
}

type Updater struct {
	queue        []Updateable
	total, empty int64
}

func (receiver *Updater) Add(object Updateable) {
	receiver.queue = append(receiver.queue, object)
	receiver.total++
}

func (receiver *Updater) Remove(object Updateable) {
	for indx, candidate := range receiver.queue {
		if object == candidate {
			receiver.queue[indx] = nil
			receiver.empty++
		}
	}
}

func (receiver *Updater) Compact() {
	i, j := 0, 0
	for i < len(receiver.queue) {
		if receiver.queue[i] == nil {
			//
		} else {
			receiver.queue[j] = receiver.queue[i]
			j++
		}
		i++
	}
	receiver.queue = receiver.queue[0:j]
	receiver.total = int64(len(receiver.queue))
	receiver.empty = 0
}

func (receiver *Updater) NeedCompact() bool {
	return receiver.total > 100 && receiver.empty > 0 && receiver.total/receiver.empty < 2
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
		queue: make([]Updateable, 0, queueSize),
	}, nil
}
