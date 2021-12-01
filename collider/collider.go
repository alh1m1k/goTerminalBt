package collider

import (
	"errors"
	"github.com/tanema/ump"
	"log"
	"os"
	"time"
)

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
				//info must be clear even if no coolision at this time
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
	/*
		reader := bufio.NewReader(os.Stdin)
		reader.ReadByte()

		bullet := cl.world.Add("bullet", 0,0,1,1)
		bullet.SetResponse("wall", "cross")
		wall := cl.world.Add("wall", 10,10, 8, 7)
		wall.SetResponse("wall", "cross")

		var start float32 = 10.0*/

	/*	_, _, collisions := bullet.Move(start + 0.000000, start + 4.099998)

		if len(collisions) > 0 {
			fmt.Println("collide")
		} else {
			fmt.Println("no collision")
		}*/

	/*	var i, y float32
		var errSeq []float32
		var gotchaCnt, okCnt = 0, 0
		for i = 0.0; i <= 8; i +=0.1 {
			for y = 0.0; y <= 7; y +=0.1 {
				_, _, gotcha := bullet.Move(start + i, start + y)
				if len(gotcha) == 0 {
					errSeq = append(errSeq, i, y)
					gotchaCnt++
				} else {
					okCnt++
				}
			}
		}
		fmt.Printf("total of %f, gotcha %d, ok %d \n", (8 / 0.1) * (7 / 0.1), gotchaCnt, okCnt)*/
	/*
		gotchaCnt = 0
		okCnt	  = 0
		for i := 0; i < len(errSeq); i+=2 {
			_, _, gotcha := bullet.Move(errSeq[i], errSeq[i+1])
			if len(gotcha) == 0 {
				gotchaCnt++
				fmt.Printf("gotcha second time %f:%f \n", errSeq[i], errSeq[i+1])
			} else {
				okCnt++
				fmt.Printf("collision %f, %f \n", errSeq[i], errSeq[i+1])
			}
		}
		fmt.Printf("second time total of %d, gotcha %d, ok %d \n", len(errSeq), gotchaCnt, okCnt)


		fmt.Printf("########\n")
		fmt.Printf("#      #\n")
		fmt.Printf("#      #\n")
		fmt.Printf("#      #\n")
		fmt.Printf("#      #\n")
		fmt.Printf("#      #\n")
		fmt.Printf("#      #\n")
		fmt.Printf("########\n")

		for i := 0; i < len(errSeq); i+=2 {
			_, _, gotcha := bullet.Move(errSeq[i], errSeq[i+1])
			if len(gotcha) == 0 {

			}
		}*/
	/*
		_, _,cl1 := bullet.Move(12,12)
		_, _,cl2 := bullet.Move(14,14)
		_, _,cl3 := bullet.Move(0,0)

		if len(cl1) > 0 && len(cl2) == 0 && len(cl3) > 0 {
			fmt.Println("done")
		}


		os.Exit(0)*/

	return cl, nil
}
