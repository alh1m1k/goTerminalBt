package main

import (
	"log"
	"math/rand"
	"os"
	"time"
)

var (
	aiBuf, _ = os.OpenFile("ai.txt", os.O_CREATE|os.O_TRUNC, 644)
	aiLogger = log.New(aiBuf, "logger: ", log.Lshortfile)
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
			tdir := control.target.Direction
			tzone := control.GetTargetZone()
			dir2zone := control.GetDirection2Zone(tzone)
			if control.InFireRange(tzone) {
				if tdir.Y != 0 && dir2zone.Y+tdir.Y == 0 {
					return true
				}
				if tdir.X != 0 && dir2zone.X+tdir.X == 0 {
					return true
				}
			}
			return false
		},
		Enter: func(control *BehaviorControl) {

		},
		Update: func(control *BehaviorControl, duration time.Duration) (done bool) {
			var weaponSolution *FireSolution

			if control.IsNeedRecalculateSolution() {
				go control.CalculateFireSolution()
				return false
			}

			azone := control.avatar.GetZone()
			tzone := control.GetTargetZone()
			tdir := control.target.Direction
			dir2zone := control.GetDirection2Zone(tzone)
			if control.InFireRange(tzone) {
				weaponSolution = control.solution
				if weaponSolution == nil {
					return
				}
				if tdir.Y != 0 && dir2zone.Y+tdir.Y == 0 {
					distance := absInt(tzone.X-azone.X) + 1 //for the width of azone
					if len(weaponSolution.sampleX) <= distance {
						return
					}
					pZone := weaponSolution.sampleX[maxInt(distance, 0)].offset
					if pZone.Y == absInt(azone.Y-tzone.Y) {
						preemption := azone
						preemption.X = preemption.X + distance*int(dir2zone.X)
						if control.AlignToZone(preemption) && control.CanHit(preemption) {
							control.Fire()
							return true
						}
					}
				} else if tdir.X != 0 && dir2zone.X+tdir.X == 0 {
					distance := absInt(tzone.Y-azone.Y) + 1 //for the height of azone
					if len(weaponSolution.sampleY) <= distance {
						return
					}
					pZone := weaponSolution.sampleY[maxInt(distance, 0)].offset
					if pZone.X == absInt(azone.X-tzone.X) {
						preemption := azone
						preemption.Y = preemption.Y + distance*int(dir2zone.Y)
						if control.AlignToZone(preemption) && control.CanHit(preemption) {
							control.Fire()
							return true
						}
					}
				}
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
			azone := control.avatar.GetZone()
			tzone := control.GetTargetZone()
			adir, tdir := control.avatar.Direction, control.target.Direction

			paralelX, paralelY := adir.X == tdir.X && tdir.X == 0, tdir.Y == adir.Y && tdir.Y == 0
			oneLineX, oneLineY := azone.X == tzone.X, azone.Y == tzone.Y

			if (paralelX && oneLineX) || (paralelY && oneLineY) && control.InFireRange(tzone) {
				if control.AlignToZone(tzone) {
					if control.CanHit(tzone) {
						control.Fire()
					}
				}
			} else {
				if OpportunityFireBehavior.Check(control) {
					OpportunityFireBehavior.Update(control, duration)
				} else {
					if PursuitBehavior.Update(control, duration) {
						///
					}
				}
			}
			return false
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
