package collider

import (
	"github.com/alh1m1k/ump"
)

type CollisionInfoSet struct {
	m map[Collideable]*ump.Collision
}

func (receiver *CollisionInfoSet) Size() int {
	return len(receiver.m)
}

func (receiver *CollisionInfoSet) Add(object Collideable, collision *ump.Collision) {
	receiver.m[object] = collision
}

func (receiver *CollisionInfoSet) I() map[Collideable]*ump.Collision {
	return receiver.m
}

func (receiver *CollisionInfoSet) Clear() {
	for index, _ := range receiver.m { //prey to https://go-review.googlesource.com/c/go/+/110055/
		delete(receiver.m, index)
	}
}

func NewCollisionInfo(size int) *CollisionInfoSet {
	return &CollisionInfoSet{m: make(map[Collideable]*ump.Collision, 5)}
}
