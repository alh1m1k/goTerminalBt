package collider

import (
	"errors"
	"github.com/alh1m1k/ump"
	"log"
	"math"
	"os"
	"time"
)

const GRID_COORD_TOLERANCE = .5

var (
	buf, _ = os.OpenFile("./collider.log", os.O_CREATE|os.O_TRUNC, 644)
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

// todo remove
func (c *Collider) AddExtra(clBody *ClBody, object Collideable) error {
	if clBody == nil {
		return errors.New("clBody was nil")
	}
	if clBody.fake {
		return nil
	}
	clBody = clBody.First
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
				body.SetResponse("perimeter", "cross")
			} else {
				body.SetResponse("penetrate", "cross")
				body.SetResponse("static", "grid")
				body.SetResponse("base", "grid")
				body.SetResponse("perimeter", "perimeter")
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

func (c *Collider) Add(object Collideable) error {
	clBody := object.GetClBody()
	if clBody == nil {
		return errors.New("clBody was nil")
	}
	clBody = clBody.First
	if clBody.collisionInfo == nil {
		clBody.collisionInfo = NewCollisionInfo(5)
	}
	if clBody.fake {
		return nil
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
				body.SetResponse("perimeter", "cross")
			} else {
				body.SetResponse("penetrate", "cross")
				body.SetResponse("static", "grid")
				body.SetResponse("base", "grid")
				body.SetResponse("perimeter", "perimeter")
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
	clBody := object.GetClBody()
	if clBody.fake {
		return
	}
	for indx, candidate := range c.bodyMap {
		if object == candidate {
			delete(c.bodyMap, indx)
			indx.Remove() //todo reenther
			clBody.realBody = nil
			clBody.First.collisionInfo.Clear()
		}
	}
}

// todo remove
func (c *Collider) RemoveExtra(clBody *ClBody, object Collideable) {
	if clBody.fake {
		return
	}
	for indx, candidate := range c.bodyMap {
		if object == candidate {
			delete(c.bodyMap, indx)
			indx.Remove() //todo reenther
			clBody.realBody = nil
			clBody.First.collisionInfo.Clear()
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

// QueryRect will take the rectangle arguments and return any bodies that are in
// that rectangle
//
// If tags are passed into the query then it will only return the bodies with those
// tags.
func (c *Collider) QueryRect(x, y, w, h float64, tags ...string) []Collideable {
	return c.filterBody(c.world.QueryRect(float32(x), float32(y), float32(w), float32(h), tags...), float32(x), float32(y), float32(w), float32(h))
}

// QueryPoint will return any bodies that are underneathe the point.
//
// If tags are passed into the query then it will only return the bodies with those
// tags.
func (c *Collider) QueryPoint(x, y float64, tags ...string) []Collideable {
	bodyList := c.world.QueryPoint(float32(x), float32(y), tags...)
	return c.bodyList2Collideable(bodyList)
}

// QuerySegment will return any bodies that are underneathe the segment/line.
//
// If tags are passed into the query then it will only return the bodies with those
// tags.
func (c *Collider) QuerySegment(x1, y1, x2, y2 float64, tags ...string) []Collideable {
	bodyList := c.world.QuerySegment(float32(x1), float32(y1), float32(x2), float32(y2), tags...)
	return c.bodyList2Collideable(bodyList)
}

// additional filtering due lib do not apply it to result :(
func (c *Collider) filterBody(bodyList []*ump.Body, x, y, w, h float32) []Collideable {
	result := make([]Collideable, 0, len(bodyList))
	for _, body := range bodyList {
		bx, by, _, _, br, bb := body.Extents()
		if br >= x && bb >= y && bx <= x+w && by <= y+h {
			if object, ok := c.bodyMap[body]; ok {
				result = append(result, object)
			}
		}
	}
	return result
}

func (c *Collider) bodyList2Collideable(bodyList []*ump.Body) []Collideable {
	result := make([]Collideable, 0, len(bodyList))
	for _, body := range bodyList {
		if object, ok := c.bodyMap[body]; ok {
			result = append(result, object)
		}
	}
	return result
}

func NewCollider(queueSize int) (*Collider, error) {
	cl := &Collider{
		bodyMap: make(map[*ump.Body]Collideable, queueSize),
		world:   ump.NewWorld(64),
		ver:     true,
	}

	cl.world.AddResponse("grid", gridFilter)
	cl.world.AddResponse("perimeter", perimeterFilter)
	cl.world.AddResponse("none", noneFilter)

	return cl, nil
}

func gridFilter(world *ump.World, col *ump.Collision, body *ump.Body, goalX, goalY float32) (float32, float32, []*ump.Collision) {
	_, _, w, h, _, _ := body.Extents()
	ox1, oy1, ow, oh, _, _ := col.Body.Extents()

	centerX := goalX + w/2
	centerY := goalY + h/2

	ocenterX := ox1 + ow/2
	ocenterY := oy1 + oh/2

	if col.Move.X != 0 { //edge case stuck on corner move
		offset := float64(centerY) - float64(ocenterY)
		distance := math.Abs(offset) - float64(h/2+oh/2)
		//if we move on x and Y in grid tolerance then ignore collide and reduce error to zero
		if distance > -GRID_COORD_TOLERANCE {
			goalY = goalY + float32(math.Copysign(distance, offset))
			body.Update(col.Touch.X, col.Touch.Y)
			return goalX, goalY, world.Project(body, goalX, goalY)
		}
	}
	if col.Move.Y != 0 {
		offset := float64(centerX) - float64(ocenterX)
		distance := math.Abs(offset) - float64(w/2+ow/2)
		if distance > -GRID_COORD_TOLERANCE {
			goalX = goalX + float32(math.Copysign(distance, offset))
			body.Update(col.Touch.X, col.Touch.Y)
			return goalX, goalY, world.Project(body, goalX, goalY)
		}
	}

	//common case collision out of tolerance
	sx, sy := col.Touch.X, col.Touch.Y
	if col.Move.X != 0 || col.Move.Y != 0 {
		if col.Normal.X == col.Normal.Y && col.Normal.X == 0 {
			logger.Printf("no normal")
		}
		if col.Normal.X == 0 {
			sx = goalX
		}
		if col.Normal.Y == 0 {
			sy = goalY
		}
	}
	col.Data = ump.Point{X: sx, Y: sy}
	body.Update(col.Touch.X, col.Touch.Y) //cause problem, prob wrong unable to use body.x, body.y = col.Touch.X, col.Touch.Y
	return sx, sy, world.Project(body, sx, sy)
}

// not work for now as no way to mark body that it already in to other ticks
func perimeterFilter(world *ump.World, col *ump.Collision, body *ump.Body, goalX, goalY float32) (float32, float32, []*ump.Collision) {
	_, _, w, h, _, _ := body.Extents()
	ox1, oy1, ow, oh, _, _ := col.Body.Extents()

	centerX := goalX + w/2
	centerY := goalY + h/2
	ocenterX := ox1 + ow/2
	ocenterY := oy1 + oh/2

	offsetX := float64(centerX) - float64(ocenterX)
	distanceX := math.Abs(offsetX) - float64(w/2+ow/2)
	offsetY := float64(centerY) - float64(ocenterY)
	distanceY := math.Abs(offsetY) - float64(h/2+oh/2)

	if distanceX < -GRID_COORD_TOLERANCE*2 && distanceY < -GRID_COORD_TOLERANCE*2 {
		body.Update(col.Touch.X, col.Touch.Y)
		return goalX, goalY, world.Project(body, goalX, goalY)
	}

	if col.Move.X != 0 {
		if distanceY > -GRID_COORD_TOLERANCE {
			goalY = goalY + float32(math.Copysign(distanceY, offsetY))
			body.Update(col.Touch.X, col.Touch.Y)
			return goalX, goalY, world.Project(body, goalX, goalY)
		}
	}
	if col.Move.Y != 0 {
		if distanceX > -GRID_COORD_TOLERANCE {
			goalX = goalX + float32(math.Copysign(distanceX, offsetX))
			body.Update(col.Touch.X, col.Touch.Y)
			return goalX, goalY, world.Project(body, goalX, goalY)
		}
	}

	sx, sy := col.Touch.X, col.Touch.Y
	if col.Move.X != 0 || col.Move.Y != 0 {
		if col.Normal.X == col.Normal.Y && col.Normal.X == 0 {
			logger.Printf("no normal")
		}
		if col.Normal.X == 0 {
			sx = goalX
		}
		if col.Normal.Y == 0 {
			sy = goalY
		}
	}
	col.Data = ump.Point{X: sx, Y: sy}
	body.Update(col.Touch.X, col.Touch.Y) //cause problem, prob wrong unable to use body.x, body.y = col.Touch.X, col.Touch.Y
	return sx, sy, world.Project(body, sx, sy)
}

/*func visionFilter(world *ump.World, col *ump.Collision, body *ump.Body, goalX, goalY float32) (float32, float32, []*ump.Collision) {
	return goalX, goalY, world.Project(body, goalX, goalY)
}*/

func noneFilter(world *ump.World, col *ump.Collision, body *ump.Body, goalX, goalY float32) (float32, float32, []*ump.Collision) {
	return goalX, goalY, []*ump.Collision{}
}
