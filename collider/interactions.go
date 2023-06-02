package collider

import (
	"github.com/alh1m1k/ump"
	"time"
)

type CollisionReceiver interface {
	OnTickCollide(object Collideable, collision *ump.Collision, owner *Interactions)
	OnStartCollide(object Collideable, collision *ump.Collision, owner *Interactions)
	OnStopCollide(object Collideable, duration time.Duration, owner *Interactions)
}

type Interactions struct {
	iteractions map[Collideable]time.Duration
	subscribers []CollisionReceiver
}

// WARNING: WATCH for composition inderect call, no check for that
func (receiver *Interactions) Subscribe(collideable CollisionReceiver) {
	receiver.subscribers = append(receiver.subscribers, collideable)
}

func (receiver *Interactions) Unsubscribe(collideable CollisionReceiver) {
	for idx, subscribe := range receiver.subscribers {
		if subscribe == collideable {
			receiver.subscribers[idx] = nil
		}
	}
}

func (receiver *Interactions) Interact(source Collideable, timeLeft time.Duration) {
	collisions := source.GetClBody().CollisionInfo().I()
	for collideable, collisionTime := range receiver.iteractions {
		if _, ok := collisions[collideable]; !ok {
			receiver.OnStopCollide(collideable, collisionTime, receiver)
			delete(receiver.iteractions, collideable)
		}
	}

	for collideable, collision := range collisions {
		if _, ok := receiver.iteractions[collideable]; ok {
			receiver.OnTickCollide(collideable, collision, receiver)
			receiver.iteractions[collideable] += timeLeft
		} else {
			receiver.OnStartCollide(collideable, collision, receiver)
			receiver.OnTickCollide(collideable, collision, receiver)
			receiver.iteractions[collideable] = timeLeft
		}
	}
}

func (receiver *Interactions) Clear() {
	for key, _ := range receiver.iteractions {
		delete(receiver.iteractions, key)
	}
	i, j := 0, 0
	for i < len(receiver.subscribers) {
		if receiver.subscribers[i] == nil {
			//
		} else {
			receiver.subscribers[j] = receiver.subscribers[i]
			j++
		}
		i++
	}
	receiver.subscribers = receiver.subscribers[0:j]
}

func (receiver *Interactions) UnsubscribeAll() {
	receiver.subscribers = receiver.subscribers[0:0]
}

func (receiver *Interactions) OnTickCollide(object Collideable, collision *ump.Collision, owner *Interactions) {
	for _, subscribe := range receiver.subscribers {
		if subscribe == nil {
			continue
		}
		subscribe.OnTickCollide(object, collision, owner)
	}
}

func (receiver *Interactions) OnStartCollide(object Collideable, collision *ump.Collision, owner *Interactions) {
	for _, subscribe := range receiver.subscribers {
		if subscribe == nil {
			continue
		}
		subscribe.OnStartCollide(object, collision, owner)
	}
}

func (receiver *Interactions) OnStopCollide(object Collideable, duration time.Duration, owner *Interactions) {
	for _, subscribe := range receiver.subscribers {
		if subscribe == nil {
			continue
		}
		subscribe.OnStopCollide(object, duration, owner)
	}
}

func (receiver *Interactions) Copy() *Interactions {
	instanse := new(Interactions)
	instanse.iteractions = make(map[Collideable]time.Duration, len(receiver.iteractions))
	instanse.subscribers = make([]CollisionReceiver, len(receiver.iteractions), len(receiver.iteractions))
	copy(instanse.subscribers, receiver.subscribers)
	for key, value := range receiver.iteractions {
		instanse.iteractions[key] = value
	}
	return instanse
}

func NewIteractions() (*Interactions, error) {
	return &Interactions{
		iteractions: make(map[Collideable]time.Duration, 10),
		subscribers: make([]CollisionReceiver, 0, 1),
	}, nil
}
