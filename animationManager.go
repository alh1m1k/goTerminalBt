package main

import (
	"sync"
	"time"
)

var globalAnimationManager *AnimationManager //for now, before blueprints

type AnimationManager struct {
	queue        []*Animation
	mutex        sync.Mutex
	total, empty int64
}

func (receiver *AnimationManager) Add(object *Animation) {
	receiver.mutex.Lock()
	receiver.queue = append(receiver.queue, object)
	object.Manager = receiver
	receiver.mutex.Unlock()
}

func (receiver *AnimationManager) Remove(object *Animation) {
	receiver.mutex.Lock()
	for indx, candidate := range receiver.queue {
		if object == candidate {
			receiver.queue[indx] = nil
			object.Manager = nil
			object.Spriteer = ErrorSprite //to visible show unmanaged but rendered animation
		}
	}
	receiver.mutex.Unlock()
}

func (receiver *AnimationManager) Compact() {
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

func (receiver *AnimationManager) NeedCompact() bool {
	return receiver.total > 100 && receiver.empty > 0 && receiver.total/receiver.empty < 2
}

func (receiver *AnimationManager) Execute(timeLeft time.Duration) {
	receiver.mutex.Lock()
	defer receiver.mutex.Unlock()
	for _, object := range receiver.queue {
		if object == nil {
			continue
		}
		object.Update(timeLeft)
	}
}

func NewAnimationManager(queueSize int) (*AnimationManager, error) {
	return &AnimationManager{
		queue: make([]*Animation, 0, queueSize),
	}, nil
}

func getAnimationManager() (*AnimationManager, error) {
	var err error
	if globalAnimationManager == nil {
		globalAnimationManager, err = NewAnimationManager(100)
	}
	return globalAnimationManager, err
}
