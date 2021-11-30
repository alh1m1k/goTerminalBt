package collider

import (
	"errors"
	"github.com/tanema/ump"
	"time"
)

type Collideable interface {
	GetClBody() *ClBody
	HasTag(tag string) bool
}


type Collider struct {
	bodyMap  map[*ump.Body]Collideable
	world    *ump.World
	ver      bool //odd even
}

func (c *Collider) Add(object Collideable) error {
	clBody := object.GetClBody()
	if clBody == nil {
		return errors.New("clBody was nil")
	}
	clBody = clBody.First
	if clBody.collisionInfo == nil {
		clBody.collisionInfo = NewCollisionInfo(5)
	}
	clBody.ver = c.ver
	for clBody != nil {
		if clBody.realBody == nil {
			x, y, w, h := clBody.GetRect()
			body := c.world.Add(clBody.filter, float32(x), float32(y), float32(w), float32(h))
			body.SetStatic(clBody.static)

			//penetrate will not affect on other object (its coordinates)
			if clBody.First.penetrate {
				body.SetResponse("penetrate", "cross")
				body.SetResponse("static", "cross")
				body.SetResponse("base", "cross")
			} else {
				body.SetResponse("penetrate", "cross")
				body.SetResponse("static", "slide")
				body.SetResponse("base", "slide")
			}

			clBody.realBody = body
		} else {
			//reenter
		}
		c.bodyMap[clBody.realBody] = object
		clBody = clBody.Next
	}
	return nil
}

func (c *Collider) Remove(object Collideable) {
	for indx, candidate := range c.bodyMap {
		if object == candidate {
			delete(c.bodyMap, indx)
			indx.Remove() //todo reenther
			object.GetClBody().realBody = nil
			if object.GetClBody().First.collisionInfo != nil {
				object.GetClBody().First.collisionInfo.Clear()
			}
		}
	}
}

func (c *Collider) Execute(timeLeft time.Duration)  {
	for realBody, object := range c.bodyMap {
		if realBody == nil {
			continue
		}
		clBody := object.GetClBody().First
		x,y := clBody.GetXY()
		for clBody != nil {
			if clBody.static {
				realBody.Update(float32(x), float32(y))
				//info must be clear even if no coolision at this time
				if clBody.First.ver != c.ver {
					clBody.First.collisionInfo.Clear()
					clBody.First.ver = c.ver
				}
			} else {
				newX, newY, collisions := realBody.Move(float32(x), float32(y))
				//todo переиспользовать clBody.collisionInfoSet
				if clBody.First.ver != c.ver {
					clBody.First.collisionInfo.Clear()
					clBody.First.ver = c.ver
				}
				for _, collision := range collisions {
					if collideWith, ok := c.bodyMap[collision.Body]; !ok {
						panic("undefined object in world!")
					} else {
						//lib generate collision only for object that exactly move, take care of that
						clBody.First.collisionInfo.Add(collideWith, collision)
						if collideWith.GetClBody().First.ver != c.ver {
							collideWith.GetClBody().collisionInfo.Clear()
							collideWith.GetClBody().First.ver = c.ver
						}
						collideWith.GetClBody().First.collisionInfo.Add(object, collision)
						//for now simplified calculation: only for first collision in list
						clBody.First.Correct(float64(newX), float64(newY)) //precision lost?
					}
				}
			}
			clBody = clBody.Next
		}
	}
	c.ver = !c.ver
}



func NewCollider(queueSize int) (*Collider, error)  {
	cl := &Collider{
		bodyMap: make(map[*ump.Body]Collideable, queueSize),
		world: ump.NewWorld(64),
		ver: true,
	}
	return cl, nil
}