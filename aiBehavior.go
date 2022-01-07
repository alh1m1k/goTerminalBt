package main

import (
	"log"
	"math"
	"math/rand"
	"os"
	"time"
)

var (
	aiBuf, _        = os.OpenFile("ai.txt", os.O_CREATE|os.O_TRUNC, 644)
	aiLogger        = log.New(aiBuf, "logger: ", log.Lshortfile)
	NoSolutionPoint = Point{
		X: -math.MaxFloat64,
		Y: -math.MaxFloat64,
	}
)

var (
	NoUpdate = func(control *BehaviorControl, duration time.Duration) (done bool) {
		return false
	}
	IdleBehavior = &Behavior{
		name: "idle",
		Check: func(control *BehaviorControl) bool {
			return true
		},
		Enter: func(control *BehaviorControl) {
			control.target = nil
			control.idle.Enable()
		},
		Update: NoUpdate,
		Leave: func(control *BehaviorControl) {
			control.idle.Disable()
		},
	}
	ChosePatternBehavior = &Behavior{
		name: "chosePatternBehavior",
		Check: func(control *BehaviorControl) bool {
			return true
		},
		Enter: func(control *BehaviorControl) {
			if control.IsNeedRecalculateSolution() {
				control.targetOffset.X = 0
				control.targetOffset.Y = 0
			} else {
				randX := triangRand(int64(len(control.solution.sampleX)))
				randY := triangRand(int64(len(control.solution.sampleY)))
				control.targetOffset.X = minInt(len(control.solution.sampleX)-int(randX), len(control.solution.sampleX))
				control.targetOffset.Y = minInt(len(control.solution.sampleY)-int(randY), len(control.solution.sampleY))
				control.targetOffset.X *= rand.Intn(2) - 1
				control.targetOffset.Y *= rand.Intn(2) - 1
				aiLogger.Print(control.targetOffset)
			}
			control.Next(HuntBehavior)
		},
		Update: NoUpdate,
		Leave: func(control *BehaviorControl) {

		},
	}
	OpportunityFireBehavior = &Behavior{
		name: "opportunityFire",
		Check: func(control *BehaviorControl) bool {
			if control.IsNeedRecalculateSolution() {
				go control.CalculateFireSolution()
				return false
			}
			target := control.target
			avatar := control.avatar

			tdir := target.Direction
			tPos := target.GetCenter2()
			aPos := avatar.GetCenter2()
			dir2Target := control.GetDirection2Target(target)

			crossing := dir2Target.Plus(tdir)
			if tdir.Y != 0 && crossing.Y == 0 && control.CanFire(Point{X: tPos.X, Y: aPos.Y}) {
				return true
			}
			if tdir.X != 0 && crossing.X == 0 && control.CanFire(Point{X: tPos.X, Y: aPos.Y}) {
				return true
			}
			return false
		},
		Enter: func(control *BehaviorControl) {

		},
		Update: func(control *BehaviorControl, duration time.Duration) (done bool) {
			const OFFSET_PRESSISION = 1.8

			var weaponSolution *FireSolution

			if control.IsNeedRecalculateSolution() {
				go control.CalculateFireSolution()
				return false
			}

			target := control.target
			avatar := control.avatar

			acenter := avatar.GetCenter2()
			aw, ah := avatar.GetWH()
			tcenter := target.GetCenter2()
			tw, th := target.GetWH()
			tdir := target.Direction
			dir2Target := control.GetDirection2Target(target)
			preemption := NoSolutionPoint

			weaponSolution = control.solution
			if weaponSolution == nil {
				return
			}
			centerDistance := acenter.Minus(tcenter).Abs()
			borderDistance := centerDistance.Minus(Center{
				X: aw/2 + tw/2,
				Y: ah/2 + th/2,
			}).Round()
			fireDistance := centerDistance.Minus(Center{
				X: tw / 2,
				Y: th / 2,
			}).Round()
			crossing := dir2Target.Plus(tdir)
			if DEBUG_OPPORTUNITY_FIRE {
				log.Print("distance/direction", borderDistance, dir2Target)
			}
			if tdir.Y != 0 && crossing.Y == 0 && control.CanFire(Point{X: tcenter.X, Y: acenter.Y}) {
				if sample := control.LookupFireSolution(weaponSolution.sampleX, borderDistance.X); sample != nil {
					offset := sample.Offset
					offset.X = fireDistance.X
					if DEBUG_OPPORTUNITY_FIRE {
						log.Print("offset", offset)
					}
					if offset.Equal(fireDistance, OFFSET_PRESSISION) {
						preemption = Point(acenter)
						preemption.X = preemption.X + dir2Target.X
					}
				}
			} else if tdir.X != 0 && crossing.X == 0 && control.CanFire(Point{X: acenter.X, Y: tcenter.Y}) {
				if sample := control.LookupFireSolution(weaponSolution.sampleY, borderDistance.Y); sample != nil {
					offset := sample.Offset
					offset.Y = fireDistance.Y
					if DEBUG_OPPORTUNITY_FIRE {
						log.Print("offset", offset)
					}
					if offset.Equal(fireDistance, OFFSET_PRESSISION) {
						preemption = Point(acenter)
						preemption.Y = preemption.Y + dir2Target.Y
					}
				}
			}
			if preemption != NoSolutionPoint {
				if control.AlignToPoint(preemption) && control.CanHit(preemption) {
					control.Fire()
				}
				return true
			}
			return false
		},
		Leave: func(control *BehaviorControl) {

		},
	}
	HuntBehavior = &Behavior{
		name: "hunt",
		Check: func(control *BehaviorControl) bool {
			return true
		},
		Enter: func(control *BehaviorControl) {
			PursuitBehavior.Enter(control)
		},
		Update: func(control *BehaviorControl, duration time.Duration) (done bool) {
			if OpportunityFireBehavior.Check(control) {
				if OpportunityFireBehavior.Update(control, duration) {
					return true
				}
			}

			apoint := control.avatar.GetCenter2()
			tpoint := control.target.GetCenter2()
			tzone := control.target.GetZone()
			adir, tdir := control.avatar.Direction, control.target.Direction
			halfCellSizeX, halfCellSizeY := control.setupUnitSize.X/2, control.setupUnitSize.Y/2

			paralelX, paralelY := adir.X == tdir.X && tdir.X == 0, tdir.Y == adir.Y && tdir.Y == 0
			oneLineX, oneLineY := math.Abs(apoint.X-tpoint.X) < halfCellSizeX, math.Abs(apoint.Y-tpoint.Y) < halfCellSizeY

			if (paralelX && oneLineX) || (paralelY && oneLineY) && control.CanFire(Point(tpoint)) {
				if control.AlignToZone(tzone) { //todo fixme
					if control.CanHit(Point(tpoint)) {
						if control.Fire() {
							return true
						}
					}
				}
			} else {
				if PursuitBehavior.Update(control, duration) {
					return true
				}
			}

			return false
		},
		Leave: func(control *BehaviorControl) {
			PursuitBehavior.Leave(control)
		},
	}
	WidrawBehavior = &Behavior{
		name: "widraw",
		Check: func(control *BehaviorControl) bool {
			return true
		},
		Enter: func(control *BehaviorControl) {
			PursuitBehavior.Enter(control)
		},
		Update: func(control *BehaviorControl, duration time.Duration) (done bool) {
			return PursuitBehavior.Update(control, duration)
		},
		Leave: func(control *BehaviorControl) {
			PursuitBehavior.Leave(control)
		},
	}
	PursuitBehavior = &Behavior{
		name: "pursuit",
		Check: func(control *BehaviorControl) bool {
			return true
		},
		Enter: func(control *BehaviorControl) {
			control.lastPath = nil
			control.newPath = nil
			PathBehavior.Enter(control)
		},
		Update: func(control *BehaviorControl, duration time.Duration) (done bool) {
			if control.lastPath == nil { //wait for the path
				if DEBUG_AI_PATH {
					logger.Printf("objectId: %d waiting for the path \n", control.avatar.ID)
				}
				return
			}
			if len(control.lastPath) > 0 {
				if control.MoveToZone(control.lastPath[0], duration) {
					control.lastPath = control.lastPath[1:]
					if len(control.lastPath) == 0 {
						if DEBUG_AI_PATH {
							logger.Printf("objectId: %d destination reach\n", control.avatar.ID)
						}
						return true
					} else {
						if DEBUG_AI_PATH {
							logger.Printf("objectId: %d %d path node left, next %d, %d \n", control.avatar.ID, len(control.lastPath), control.lastPath[0].X, control.lastPath[0].Y)
						}
						control.AlignToZone(control.lastPath[0]) //direction to new zone
					}
				}
			}
			return false
		},
		Leave: func(control *BehaviorControl) {

			//todo go to .Next only after update

			PathBehavior.Leave(control)
			control.lastPath = nil
			control.newPath = nil
		},
	}
	PathBehavior = &Behavior{
		name: "path",
		Check: func(control *BehaviorControl) bool {
			return true
		},
		Enter: func(control *BehaviorControl) {
			control.target.GetTracker().Subscribe(control)
			control.OnIndexUpdate(nil)
		},
		Update: NoUpdate,
		Leave: func(control *BehaviorControl) {
			control.target.GetTracker().Unsubscribe(control)
		},
	}
)

type BehaviorInterface interface {
	Name() string
	Check(control *BehaviorControl) bool
	Enter(control *BehaviorControl)
	Update(control *BehaviorControl, duration time.Duration) (done bool)
	Leave(control *BehaviorControl)
}

type Behavior struct {
	name   string
	Check  func(control *BehaviorControl) bool
	Enter  func(control *BehaviorControl)
	Update func(control *BehaviorControl, duration time.Duration) (done bool)
	Leave  func(control *BehaviorControl)
}

func (receiver *Behavior) Name() string {
	return receiver.name
}
