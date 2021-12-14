package main

import (
	"GoConsoleBT/collider"
	"sync"
	"time"
)

type Seen interface {
	GetVision() *collider.ClBody
}

type Visioner struct {
	collider     *collider.Collider
	queue        []Seen
	mutex        sync.Mutex
	total, empty int64
}

func (receiver *Visioner) Add(object Seen) {
	if v := object.GetVision(); v == nil {
		return
	}
	receiver.mutex.Lock()
	receiver.queue = append(receiver.queue, object)
	receiver.mutex.Unlock()
}

func (receiver *Visioner) Remove(object Seen) {
	receiver.mutex.Lock()
	for indx, candidate := range receiver.queue {
		if object == candidate {
			receiver.queue[indx] = nil
			object.GetVision().CollisionInfo().Clear()
		}
	}
	receiver.mutex.Unlock()
}

func (receiver *Visioner) Compact() {
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
	receiver.queue = receiver.queue[0 : j+1]
	receiver.total = int64(len(receiver.queue))
	receiver.empty = 0
}

func (receiver *Visioner) NeedCompact() bool {
	return receiver.total > 100 && receiver.empty > 0 && receiver.total/receiver.empty < 2
}

func (receiver *Visioner) Execute(timeLeft time.Duration) {
	receiver.mutex.Lock()
	defer receiver.mutex.Unlock()
	for _, object := range receiver.queue {
		if object == nil {
			continue
		}
		vision := object.GetVision()
		vision.CollisionInfo().Clear()
		for _, qObject := range receiver.collider.QueryRect(vision.GetRect()) {
			/*			if &object == qObject {
						continue
					}*/
			vision.CollisionInfo().Add(qObject, nil)
		}
	}
}

func NewVisioner(collider *collider.Collider, queueSize int) (*Visioner, error) {
	return &Visioner{
		collider: collider,
		queue:    make([]Seen, 0, queueSize),
	}, nil
}
