package main

import (
	"sync"
	"time"
)

var globalAnimationManager *AnimationManager //for now, before blueprints

type AnimationManager struct {
	queue []*Animation
	mutex sync.Mutex
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
		}
	}
	receiver.mutex.Unlock()
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
