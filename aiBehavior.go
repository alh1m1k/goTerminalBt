package main

import (
	"GoConsoleBT/collider"
	"GoConsoleBT/controller"
	"context"
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
	NoOp = func(control *BehaviorControl) {}
	OkOp = func(control *BehaviorControl) bool {
		return true
	}

	IdleBehavior = &Behavior{
		name:  "idle",
		Check: OkOp,
		Enter: func(control *BehaviorControl) {
			control.target = nil
			control.idle.Enable()
		},
		Update: func(control *BehaviorControl, duration time.Duration) (done bool) {
			if control.blockedDirection[control.avatar.Direction] {
				if CycleID%3 == 0 {
					go control.Fire()
					return false
				}
				for dir, blocked := range control.blockedDirection {
					if !blocked {
						go func() { //todo remove
							control.commandChanel <- controller.Command{
								CType:  controller.CTYPE_DIRECTION,
								Pos:    controller.Point(dir),
								Action: true,
							}
						}()
						break
					}
				}
			}
			return false
		},
		Leave: func(control *BehaviorControl) {
			control.idle.Disable()
		},
		Next: NoOp,
	}
	ChosePatternBehavior = &Behavior{
		name:  "chose",
		Check: OkOp,
		Enter: func(control *BehaviorControl) {
			if len(control.availableTargets) > 0 {
				var base, unit *Unit
				control.target = nil
				for _, targetCandidate := range control.availableTargets {
					if targetCandidate == nil || targetCandidate.destroyed {
						continue
					}
					if targetCandidate.HasTag("base") && base == nil {
						base = targetCandidate
					} else if unit == nil {
						unit = targetCandidate
					}
				}
				if base != nil && unit != nil {
					if rand.Intn(3) >= 2 {
						control.target = base
					} else {
						control.target = unit
					}
				} else {
					if base != nil {
						control.target = base
					} else {
						control.target = unit
					}
				}
			}
			if control.target == nil {
				control.Next(IdleBehavior)
				return
			}
			if control.IsNeedRecalculateSolution() {
				control.targetOffset.X = 0
				control.targetOffset.Y = 0
			} else {
				/*				randX := triangRand(int64(len(control.solution.sampleX)))
								randY := triangRand(int64(len(control.solution.sampleY)))
								control.targetOffset.X = minInt(len(control.solution.sampleX)-int(randX), len(control.solution.sampleX))
								control.targetOffset.Y = minInt(len(control.solution.sampleY)-int(randY), len(control.solution.sampleY))
								control.targetOffset.X *= rand.Intn(2) - 1
								control.targetOffset.Y *= rand.Intn(2) - 1*/
				//aiLogger.Print(control.targetOffset)
			}
			if control.target.HasTag("base") {
				control.Next(NewSiegeBehavior())
			} else {
				control.Next(NewHuntBehavior())
			}
		},
		Update: NoUpdate,
		Leave:  NoOp,
		Next:   NoOp,
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
		Next: NoOp,
	}
)

func NewHuntBehavior() *Behavior {
	pursuitBehavior := NewPursuitBehavior()
	return &Behavior{
		name:  "hunt",
		Check: OkOp,
		Enter: func(control *BehaviorControl) {
			pursuitBehavior.Enter(control)
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
			cPrecX, cPrecY := control.setupUnitSize.X/4, control.setupUnitSize.Y/4

			paralelX, paralelY := adir.X == tdir.X && tdir.X == 0, tdir.Y == adir.Y && tdir.Y == 0
			oneLineX, oneLineY := math.Abs(apoint.Y-tpoint.Y) < cPrecY, math.Abs(apoint.X-tpoint.X) < cPrecX

			if len(control.lastPath) == 0 && oneLineX || oneLineY {
				asize, tsize := control.avatar.GetWH2(), control.target.GetWH2()
				distance := apoint.Minus(tpoint).Abs().Minus(Center{X: asize.W/2 + tsize.W/2, Y: asize.H/2 + tsize.H/2})
				if (distance.X < asize.W*2 && oneLineX) || (distance.Y < asize.H*2 && oneLineY) {
					current := control.Behavior
					control.Next(NewAttackBehavior(func(control *BehaviorControl) {
						control.Next(current)
					}))
				}
			}

			if (paralelX && oneLineX) || (paralelY && oneLineY) && control.CanFire(Point(tpoint)) {
				if control.AlignToZone(tzone) { //todo fixme
					if control.CanHit(Point(tpoint)) {
						if control.Fire() {
							return true
						}
					}
				}
			} else {
				if pursuitBehavior.Update(control, duration) {
					return true
				}
			}

			return false
		},
		Leave: func(control *BehaviorControl) {
			pursuitBehavior.Leave(control)
		},
		Next: NoOp,
	}
}
func NewSiegeBehavior() *Behavior {
	behavior := NewHuntBehavior()
	oldEnter := behavior.Enter
	oldLeave := behavior.Leave
	var oldOffset Zone
	behavior.name = "siege"
	behavior.Enter = func(control *BehaviorControl) {
		oldOffset = control.targetOffset
		control.targetOffset.X, control.targetOffset.Y = 0, 0 //hardcode
		switch rand.Intn(3) {
		case 0: //left
			control.targetOffset.X -= 2
		case 1: //top
			control.targetOffset.Y -= 2
		case 2: //right
			control.targetOffset.X = 2
		}
		oldEnter(control)
	}
	behavior.Leave = func(control *BehaviorControl) {
		oldLeave(control)
		control.targetOffset = oldOffset
	}
	return behavior
}
func NewPursuitBehavior() *Behavior {
	pathBehavior := NewPathBehavior(time.Second)
	return &Behavior{
		name: "pursuit",
		Check: func(control *BehaviorControl) bool {
			return true
		},
		Enter: func(control *BehaviorControl) {
			control.lastPath = nil
			control.newPath = nil
			pathBehavior.Enter(control)
		},
		Update: func(control *BehaviorControl, duration time.Duration) (done bool) {
			if control.lastPath == nil { //wait for the path
				if DEBUG_AI_PATH {
					logger.Printf("objectId: %d waiting for the path \n", control.avatar.ID)
				}
				return
			}
			if len(control.lastPath) > 0 {
				if done, err := control.MoveToZone(control.lastPath[0], duration); done {
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
				} else if err == MoveBlockedError {
					if DEBUG_AI_PATH {
						logger.Printf("cycleId: %d, objectId: %d moving blocked -> %t, %t\n", CycleID, control.avatar.ID, control.avatar.GetZone(), control.lastPath[0])
					}
					control.Next(NewIdleUntilBehavior(func(control *BehaviorControl) bool {
						return !control.IsFullBlock()
					}, control.Behavior, true))
				}
			} else {
				if control.noPath {
					current := control.Behavior
					if control.CountBlockedDirection() >= 3 {
						idle := NewIdleUntilBehavior(func(control *BehaviorControl) bool {
							return !control.IsFullBlock() && !control.noPath
						}, current, true)
						clear := NewClearPathBehavior(func(control *BehaviorControl) {
							if control.CountBlockedDirection() >= 3 {
								control.Next(idle)
							} else {
								control.Next(current)
							}
						}, 3)
						if clear.Check(control) { //clearable
							control.Next(clear)
						} else {
							control.Next(idle)
						}
					} else {
						idle := NewIdleUntilBehavior(func(control *BehaviorControl) bool {
							return !control.IsFullBlock() && !control.noPath
						}, control.Behavior, true)
						control.Next(idle)
					}
					return true
				}
			}
			return false
		},
		Leave: func(control *BehaviorControl) {
			pathBehavior.Leave(control)
			control.lastPath = nil
			control.newPath = nil
		},
		Next: NoOp,
	}
}
func NewPathBehavior(updateDl time.Duration) *Behavior {
	ctx, cancelFunc := context.WithCancel(context.TODO())
	return &Behavior{
		name:  "path",
		Check: OkOp,
		Enter: func(control *BehaviorControl) {
			control.target.GetTracker().Subscribe(control)
			control.OnIndexUpdate(nil)
			everyFunc(updateDl, func() { //todo respect lastUpdate time
				control.OnIndexUpdate(nil)
			}, ctx)
		},
		Update: NoUpdate,
		Leave: func(control *BehaviorControl) {
			control.target.GetTracker().Unsubscribe(control)
			cancelFunc()
		},
		Next: NoOp,
	}
}
func NewIdleUntilBehavior(check func(control *BehaviorControl) bool, toBehavior *Behavior, keepTrack bool) *Behavior {
	return &Behavior{
		name:  "idleUntil",
		Check: OkOp,
		Enter: func(control *BehaviorControl) {
			control.idle.Enable()
			if keepTrack {
				control.target.GetTracker().Subscribe(control)
				control.OnIndexUpdate(nil)
			}
		},
		Update: func(control *BehaviorControl, duration time.Duration) (done bool) {
			if check(control) {
				control.Next(toBehavior)
				return true
			} else {
				return IdleBehavior.Update(control, duration)
			}
		},
		Leave: func(control *BehaviorControl) {
			control.target.GetTracker().Unsubscribe(control)
		},
		Next: NoOp,
	}
}
func NewMove2Behavior(zone Zone, toBehavior *Behavior, keepTrack bool) *Behavior {
	return &Behavior{
		name:  "move2",
		Check: OkOp,
		Enter: func(control *BehaviorControl) {
			control.lastPath = nil
			control.newPath = nil
			control.Navigation.SchedulePath(control.avatar.GetZone(), zone, control)
		},
		Update: func(control *BehaviorControl, duration time.Duration) (done bool) {
			if control.lastPath == nil { //wait for the path
				if DEBUG_AI_PATH {
					logger.Printf("objectId: %d waiting for the path \n", control.avatar.ID)
				}
				return
			}
			if len(control.lastPath) > 0 {
				if done, err := control.MoveToZone(control.lastPath[0], duration); done {
					control.lastPath = control.lastPath[1:]
					if len(control.lastPath) == 0 {
						if DEBUG_AI_PATH {
							logger.Printf("objectId: %d destination reach\n", control.avatar.ID)
						}
						control.Next(toBehavior)
					} else {
						if DEBUG_AI_PATH {
							logger.Printf("objectId: %d %d path node left, next %d, %d \n", control.avatar.ID, len(control.lastPath), control.lastPath[0].X, control.lastPath[0].Y)
						}
						control.AlignToZone(control.lastPath[0]) //direction to new zone
					}
				} else if err == MoveBlockedError {
					logger.Printf("objectId: %d moving blocked -> %t, %t\n", control.avatar.ID, control.avatar.GetZone(), control.lastPath[0])
					control.Next(NewIdleUntilBehavior(func(control *BehaviorControl) bool {
						return !control.IsFullBlock()
					}, control.Behavior, true))
				}
			} else {
				if control.noPath {
					control.Next(NewIdleUntilBehavior(func(control *BehaviorControl) bool {
						return !control.IsFullBlock() && !control.noPath
					}, control.Behavior, true))
					return true
				}
			}
			return false
		},
		Leave: func(control *BehaviorControl) {
			control.target.GetTracker().Unsubscribe(control)
		},
		Next: NoOp,
	}
}
func NewEvadeBehavior(check func(control *BehaviorControl) bool, toBehavior *Behavior, keepTrack bool) *Behavior {
	return &Behavior{
		name:  "evade",
		Check: OkOp,
		Enter: func(control *BehaviorControl) {
			control.idle.Enable()
			if keepTrack {
				control.target.GetTracker().Subscribe(control)
				control.OnIndexUpdate(nil)
			}
		},
		Update: func(control *BehaviorControl, duration time.Duration) (done bool) {
			if check(control) {
				control.Next(toBehavior)
				return true
			} else {
				return false
			}
		},
		Leave: func(control *BehaviorControl) {
			control.target.GetTracker().Unsubscribe(control)
		},
		Next: NoOp,
	}
}
func NewClearPathBehavior(next func(control *BehaviorControl), blockedThreshold int8) *Behavior {
	blockedThreshold = int8(minInt(maxInt(0, int(blockedThreshold)), 4))
	return &Behavior{
		name: "clearPath",
		Check: func(control *BehaviorControl) bool {
			return control.CountBlockedDirection() >= blockedThreshold
		},
		Enter: NoOp,
		Update: func(control *BehaviorControl, duration time.Duration) (done bool) {
			if control.CountBlockedDirection() < blockedThreshold {
				return true
			}
			candidates := control.blockerMap[control.target.Direction]
			var target collider.Collideable
			var direction Point = NoPos
			for _, candidate := range candidates {
				if candidate.HasTag("vulnerable") && !candidate.HasTag("explosive") {
					target = candidate
					direction = control.target.Direction
				}
			}
			if target != nil {
				for cdir, candidates := range control.blockerMap {
					for _, candidate := range candidates {
						if candidate.HasTag("vulnerable") && !candidate.HasTag("explosive") {
							target = candidate
							direction = cdir
						}
					}
				}
			}

			if target != nil && direction != NoPos {
				x, y := target.GetClBody().GetCenter()
				point := Point{x, y}
				if control.AlignToDirection(direction) {
					if control.CanFire(point) && control.CanHit(point) {
						control.Fire()
					}
				}
			} else {
				return true
			}
			return false
		},
		Leave: NoOp,
		Next:  next,
	}
}
func NewAttackBehavior(next func(control *BehaviorControl)) *Behavior {
	return &Behavior{
		name:  "attack",
		Check: OkOp,
		Enter: NoOp,
		Update: func(control *BehaviorControl, duration time.Duration) (done bool) {
			if control.target == nil || control.target.destroyed {
				return true
			}
			if !control.IsStop() {
				control.Stop()
			}
			apoint := control.avatar.GetCenter2()
			tpoint := control.target.GetCenter2()
			tzone := control.target.GetZone()
			cPrecX, cPrecY := control.setupUnitSize.X/4, control.setupUnitSize.Y/4

			oneLineX, oneLineY := math.Abs(apoint.Y-tpoint.Y) < cPrecY, math.Abs(apoint.X-tpoint.X) < cPrecX

			asize, tsize := control.avatar.GetWH2(), control.target.GetWH2()
			distance := apoint.Minus(tpoint).Abs().Minus(Center{X: asize.W/2 + tsize.W/2, Y: asize.H/2 + tsize.H/2})
			if (distance.X < asize.W*2 && oneLineX) || (distance.Y < asize.H*2 && oneLineY) {
				if control.CanFire(Point(tpoint)) {
					if control.AlignToZone(tzone) { //todo fixme
						if control.CanHit(Point(tpoint)) {
							if control.Fire() {
								return false
							}
						}
					}
				}
			} else {
				return true
			}

			return false
		},
		Leave: NoOp,
		Next:  next,
	}
}
func NewWithdrawalBehavior() *Behavior {
	pursuitBehavior := NewPursuitBehavior()
	return &Behavior{
		name:  "widraw",
		Check: OkOp,
		Enter: func(control *BehaviorControl) {
			pursuitBehavior.Enter(control)
		},
		Update: func(control *BehaviorControl, duration time.Duration) (done bool) {
			return pursuitBehavior.Update(control, duration)
		},
		Leave: func(control *BehaviorControl) {
			pursuitBehavior.Leave(control)
		},
		Next: NoOp,
	}
}

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
	Next   func(control *BehaviorControl)
}

func (receiver *Behavior) Name() string {
	return receiver.name
}
