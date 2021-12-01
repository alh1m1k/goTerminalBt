package collider

import (
	"github.com/tanema/ump"
	"time"
)

type CollisionReceiver interface {
	OnTickCollide(object Collideable, collision *ump.Collision)
	OnStartCollide(object Collideable, collision *ump.Collision)
	OnStopCollide(object Collideable, duration time.Duration)
}

type Interactions struct {
	iteractions map[Collideable]time.Duration
	subscribers []CollisionReceiver
}

//WARNING: WATCH for composition inderect call, no check for that
func (receiver *Interactions) Subscribe(collideable CollisionReceiver) {
	receiver.subscribers = append(receiver.subscribers, collideable)
}

func (receiver *Interactions) Interact(source Collideable, timeLeft time.Duration) {
	collisions := source.GetClBody().CollisionInfo().I()
	for collideable, collisionTime := range receiver.iteractions {
		if _, ok := collisions[collideable]; !ok {
			receiver.OnStopCollide(collideable, collisionTime)
			delete(receiver.iteractions, collideable)
		}
	}

	for collideable, collision := range collisions {
		if _, ok := receiver.iteractions[collideable]; ok {
			receiver.OnTickCollide(collideable, collision)
			receiver.iteractions[collideable] += timeLeft
		} else {
			receiver.OnStartCollide(collideable, collision)
			receiver.OnTickCollide(collideable, collision)
			receiver.iteractions[collideable] = timeLeft
		}
	}
}

func (receiver *Interactions) Clear() {
	for key, _ := range receiver.iteractions {
		delete(receiver.iteractions, key)
	}
}

func (receiver *Interactions) OnTickCollide(object Collideable, collision *ump.Collision) {
	for _, subscribe := range receiver.subscribers {
		subscribe.OnTickCollide(object, collision)
	}
}

func (receiver *Interactions) OnStartCollide(object Collideable, collision *ump.Collision) {
	for _, subscribe := range receiver.subscribers {
		subscribe.OnStartCollide(object, collision)
	}
}

func (receiver *Interactions) OnStopCollide(object Collideable, duration time.Duration) {
	for _, subscribe := range receiver.subscribers {
		subscribe.OnStopCollide(object, duration)
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
