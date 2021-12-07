package collider

import (
	"errors"
	"github.com/tanema/ump"
	"log"
	"math"
	"os"
	"time"
)

const GRID_COORD_TOLERANCE = 1

var (
	buf, _ = os.OpenFile("collider.log", os.O_CREATE|os.O_TRUNC, 644)
	logger = log.New(buf, "logger: ", log.Lshortfile)
)

type Collideable interface {
	GetClBody() *ClBody
	HasTag(tag string) bool
}

type Collider struct {
	bodyMap map[*ump.Body]Collideable
	world   *ump.World
	ver     bool //odd even
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
				body.SetResponse("static", "grid")
				body.SetResponse("base", "grid")
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

func (c *Collider) Execute(timeLeft time.Duration) {
	for realBody, object := range c.bodyMap {
		if realBody == nil {
			continue
		}
		clBody := object.GetClBody().First
		x, y := clBody.GetXY()
		if clBody.ver != c.ver {
			clBody.collisionInfo.Clear()
			clBody.ver = c.ver
		}
		for clBody != nil {
			if clBody.static {
				realBody.Update(float32(x), float32(y))
				//info must be clear even if no collision at this time
			} else {
				newX, newY, collisions := realBody.Move(float32(x), float32(y))
				for _, collision := range collisions {
					if collideWith, ok := c.bodyMap[collision.Body]; !ok {
						panic("undefined object in world!")
					} else {
						//lib generate collision only for object that exactly move, take care of that
						clBody.First.collisionInfo.Add(collideWith, collision)
						if collideWith.GetClBody().First.ver != c.ver {
							collideWith.GetClBody().First.collisionInfo.Clear()
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

func NewCollider(queueSize int) (*Collider, error) {
	cl := &Collider{
		bodyMap: make(map[*ump.Body]Collideable, queueSize),
		world:   ump.NewWorld(64),
		ver:     true,
	}

	cl.world.AddResponse("grid", gridFilter)

	//reader := bufio.NewReader(os.Stdin)
	/*
		cl.world.AddResponse("grid", gridFilter)
		bullet := cl.world.Add("base", 0,20.00,9.998,9.998)
		bullet.SetResponse("wall", "grid")
		wall := cl.world.Add("wall", 100,10, 10, 10)
		wall2 := cl.world.Add("wall", 100,30, 10, 10)
		wall.SetResponse("wall", "grid")
		wall2.SetResponse("wall", "grid") */

	return cl, nil
}

func gridFilter(world *ump.World, col *ump.Collision, body *ump.Body, goalX, goalY float32) (float32, float32, []*ump.Collision) {
	_, _, w, h, _, _ := body.Extents()
	ox1, oy1, ow, oh, _, _ := col.Body.Extents()

	centerX := goalX + w/2
	centerY := goalY + h/2

	ocenterX := ox1 + ow/2
	ocenterY := oy1 + oh/2

	if col.Move.X != 0 {
		offset := float64(centerY) - float64(ocenterY)
		distance := math.Abs(offset) - float64(h/2+oh/2)
		if distance > -GRID_COORD_TOLERANCE {
			goalY = goalY + float32(math.Copysign(distance, offset))
			body.Update(goalX, goalY)
			return goalX, goalY, world.Project(body, goalX, goalY)
		}
	}

	if col.Move.Y != 0 {
		offset := float64(centerX) - float64(ocenterX)
		distance := math.Abs(offset) - float64(w/2+ow/2)
		if distance > -GRID_COORD_TOLERANCE {
			goalX = goalX + float32(math.Copysign(distance, offset))
			body.Update(goalX, goalY)
			return goalX, goalY, world.Project(body, goalX, goalY)
		}
	}

	sx, sy := col.Touch.X, col.Touch.Y
	if col.Move.X != 0 || col.Move.Y != 0 {
		if col.Normal.X == 0 {
			sx = goalX
		} else {
			sy = goalY
		}
	}
	col.Data = ump.Point{X: sx, Y: sy}
	body.Update(col.Touch.X, col.Touch.Y)
	return sx, sy, world.Project(body, sx, sy)
}
