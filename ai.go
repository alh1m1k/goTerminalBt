package main

import (
	"GoConsoleBT/collider"
	"GoConsoleBT/controller"
	"context"
	"errors"
	"github.com/tanema/ump"
	"math"
	"sync"
	"sync/atomic"
	"time"
)

var (
	UndefinedProjectileError = errors.New("undefined projectile")
	MoveBlockedError         = errors.New("moving is blocked")
	bottom                   = Point{0, 1}
	top                      = Point{0, -1}
	right                    = Point{1, 0}
	left                     = Point{-1, 0}
)

type BehaviorControlBuilder struct {
	Builder
	*collider.Collider
	*Location
	*Navigation
	projectileProto map[string]*Projectile
}

func (receiver *BehaviorControlBuilder) RegisterProjectile(projectile *Projectile) error {

	receiver.projectileProto[projectile.GetAttr().Blueprint] = projectile

	return nil
}

func (receiver *BehaviorControlBuilder) Build() (*BehaviorControl, error) {
	instance, _ := NewAIControl()
	instance.Collider = receiver.Collider
	instance.Location = receiver.Location
	instance.Navigation = receiver.Navigation
	instance.projectileProto = receiver.projectileProto
	instance.builder = receiver
	return instance, nil
}

type BehaviorControl struct {
	*controller.Control
	idle *controller.Control
	*collider.Collider
	*Location
	*Navigation
	*Behavior
	builder                      *BehaviorControlBuilder
	nextBehavior                 *Behavior
	avatar                       *Unit
	target                       *Unit
	availableTargets             []*Unit
	targetOffset                 Zone
	lastPath, newPath            []Zone
	newPathId                    int64
	disabled, solutionCalculated bool
	solution                     *FireSolution
	projectileProto              map[string]*Projectile
	commandChanel                chan controller.Command
	pathLock                     sync.Mutex
	blockedDirection             map[Point]bool
	blockerMap                   map[Point][]collider.Collideable
	pathCalculated, noPath       bool
	aiCtx                        context.Context
}

func (receiver *BehaviorControl) AttachTo(object *Unit) {
	if receiver.avatar == object {
		return
	}
	if receiver.avatar != nil {
		receiver.Deattach()
	}
	receiver.avatar = object

	if !receiver.disabled {
		receiver.attach(object)
		receiver.Next(IdleBehavior)
	}
}

func (receiver *BehaviorControl) Deattach() {
	receiver.Next(IdleBehavior)
	receiver.deattach()
	receiver.Disable()
	receiver.avatar = nil
}

func (receiver *BehaviorControl) GetCommandChanel() controller.CommandChanel {
	return receiver.commandChanel
}

func (receiver *BehaviorControl) See(object *Unit) {
	if receiver.target != nil && receiver.target.destroyed {
		if DEBUG_AI_BEHAVIOR {
			logger.Printf("object id %d reset target due it destruction")
		}
		receiver.forget(receiver.target)
		receiver.target = nil
	}

	if receiver.target != nil {
		//todo target selection
		if receiver.target == object {
			if DEBUG_AI_BEHAVIOR {
				logger.Printf("object id %d skip new target because target is same", receiver.avatar.ID, object.ID)
			}
		} else {
			receiver.memorize(object)
			if DEBUG_AI_BEHAVIOR {
				logger.Printf("object id %d already have target but memorize new also", receiver.avatar.ID, object.ID)
			}
		}
	} else {
		receiver.memorize(object)
		if DEBUG_AI_BEHAVIOR {
			logger.Printf("object id %d see object id %d", receiver.avatar.ID, object.ID)
		}
	}
	receiver.Next(ChosePatternBehavior)
}

func (receiver *BehaviorControl) UnSee(object *Unit) {
	if DEBUG_AI_BEHAVIOR {
		logger.Printf("object id %d unsee object id %d", receiver.avatar.ID, object.ID)
	}
	receiver.forget(object)
	receiver.Next(ChosePatternBehavior)
}

func (receiver *BehaviorControl) UnSeeAll() {
	receiver.Next(IdleBehavior)
}

func (receiver *BehaviorControl) Enable() error {
	receiver.idle.Enable()
	if receiver.avatar != nil {
		receiver.attach(receiver.avatar)
	}
	if receiver.IsNeedRecalculateSolution() {
		receiver.CalculateFireSolution()
	}
	receiver.Next(IdleBehavior)
	return nil
}

func (receiver *BehaviorControl) Disable() error {
	receiver.idle.Disable()
	if receiver.Behavior != nil {
		receiver.Behavior.Leave(receiver)
	}
	receiver.UnSeeAll()
	receiver.deattach()
	return nil
}

func (receiver *BehaviorControl) Update(timeLeft time.Duration) error {
	if receiver.target != nil && receiver.target.destroyed {
		receiver.UnSee(receiver.target)
	}
	if path := receiver.newPath; path != nil {
		if len(path) == 0 {
			if DEBUG_AI_PATH {
				logger.Printf("cycleId: %d, objectId: %d receive zero path to target", CycleID, receiver.avatar.ID)
			}
			receiver.noPath = true
		} else {
			receiver.noPath = false
		}
		if len(receiver.lastPath) > 0 {
			receiver.lastPath = receiver.cutoffPath(path, receiver.lastPath[0])
		} else {
			receiver.lastPath = path
		}
		receiver.newPath = nil
		receiver.pathCalculated = true
	}
	if behavior := receiver.nextBehavior; behavior != nil {
		receiver.nextBehavior = nil
		receiver.next(behavior)
	}
	if receiver.Behavior != nil {
		if receiver.Behavior.Update(receiver, timeLeft) {
			receiver.Behavior.Next(receiver)
		}
	}

	receiver.resetBlockerList()
	return nil
}

func (receiver *BehaviorControl) resetBlockerList() {
	receiver.blockedDirection[right], receiver.blockerMap[right] = false, receiver.blockerMap[right][0:0]
	receiver.blockedDirection[left], receiver.blockerMap[left] = false, receiver.blockerMap[left][0:0]
	receiver.blockedDirection[top], receiver.blockerMap[top] = false, receiver.blockerMap[top][0:0]
	receiver.blockedDirection[bottom], receiver.blockerMap[bottom] = false, receiver.blockerMap[bottom][0:0]
}

//todo return to
func (receiver *BehaviorControl) Next(behavior *Behavior) {
	if receiver.Behavior != nil && &receiver.Behavior.Update == &NoUpdate {
		panic("atatat")
	} else {
		if receiver.nextBehavior != behavior {
			if DEBUG_AI_BEHAVIOR {
				logger.Printf("cycleId: %d, objectId: %d Shedule next behavior %s", CycleID, receiver.avatar.ID, behavior.Name())
			}
			if receiver.nextBehavior != nil && receiver.nextBehavior.Name() == "chose" {
				logger.Print("BUG# attempt to override chose bh")
				return
			}
			receiver.nextBehavior = behavior
		}
	}
}

func (receiver *BehaviorControl) Copy() controller.Controller {
	instance, _ := receiver.builder.Build()
	return instance
}

func (receiver *BehaviorControl) next(behavior *Behavior) {
	if receiver.Behavior != nil {
		if DEBUG_AI_BEHAVIOR {
			logger.Printf("cycleId: %d, objectId: %d behavior %s -> %s", CycleID, receiver.avatar.ID, receiver.Behavior.Name(), behavior.Name())
		}
		receiver.Behavior.Leave(receiver)
	} else {
		if DEBUG_AI_BEHAVIOR {
			logger.Printf("cycleId: %d, objectId: %d behavior %s", CycleID, receiver.avatar.ID, behavior.Name())
		}
	}
	receiver.Behavior = behavior
	receiver.Behavior.Enter(receiver)
}

func (receiver *BehaviorControl) IsReachable(zone Zone) bool {
	return true
}

func (receiver *BehaviorControl) InZone(zone Zone) bool {
	if receiver.avatar == nil || receiver.avatar.GetTracker() == nil {
		return false
	}
	center := receiver.avatar.GetCenter2()
	centerOfZone, _ := receiver.Location.CenterByIndex(zone.X, zone.Y)
	if center == centerOfZone {
		return true
	}
	if math.Abs(centerOfZone.X-center.X) <= 0.1 && math.Abs(centerOfZone.Y-center.Y) <= 0.1 {
		return true
	}
	return false
}

func (receiver *BehaviorControl) IsAlignToZone(zone Zone) bool {
	//todo cache center pos in zone
	center := receiver.avatar.GetCenter2()
	centerOfZone, _ := receiver.Location.CenterByIndex(zone.X, zone.Y)

	var delta float64
	var direction Point
	if delta = centerOfZone.Y - center.Y; math.Abs(delta) > 0.1 {
		if delta > 0 {
			direction.Y = 1
		} else {
			direction.Y = -1
		}
	}
	if delta = centerOfZone.X - center.X; math.Abs(delta) > 0.1 {
		if delta > 0 {
			direction.X = 1
		} else {
			direction.X = -1
		}
	}

	return receiver.avatar.Direction == direction
}

func (receiver *BehaviorControl) AlignToZone(zone Zone) (done bool) {
	//todo cache center pos in zone
	if receiver.IsStop() {

	}
	center := receiver.avatar.GetCenter2()
	centerOfZone, _ := receiver.Location.CenterByIndex(zone.X, zone.Y)

	moveCommand := controller.Command{
		CType:  controller.CTYPE_DIRECTION,
		Pos:    controller.Point{},
		Action: true,
	}

	var delta float64
	if delta = centerOfZone.Y - center.Y; math.Abs(delta) > 0.1 {
		if delta > 0 {
			moveCommand.Pos.Y = 1
		} else {
			moveCommand.Pos.Y = -1
		}
	} else if delta = centerOfZone.X - center.X; math.Abs(delta) > 0.1 {
		if delta > 0 {
			moveCommand.Pos.X = 1
		} else {
			moveCommand.Pos.X = -1
		}
	}

	if receiver.avatar.Direction == Point(moveCommand.Pos) {
		return true
	}

	receiver.commandChanel <- moveCommand

	return false
}

func (receiver *BehaviorControl) IsAlignToDirection(direction Point) bool {
	return receiver.avatar.Direction == direction
}

func (receiver *BehaviorControl) AlignToDirection(direction Point) (done bool) {

	if receiver.avatar.Direction == direction {
		return true
	}

	moveCommand := controller.Command{
		CType: controller.CTYPE_DIRECTION,
		Pos:   controller.Point(direction),
	}

	receiver.commandChanel <- moveCommand

	return false
}

func (receiver *BehaviorControl) IsAlignToPoint(direction Point) bool {
	return receiver.avatar.Direction == direction
}

func (receiver *BehaviorControl) AlignToPoint(point Point) (done bool) {

	direction := receiver.GetDirection2Point(point)

	if receiver.avatar.Direction == direction {
		return true
	}

	moveCommand := controller.Command{
		CType: controller.CTYPE_DIRECTION,
		Pos:   controller.Point(direction),
	}

	receiver.commandChanel <- moveCommand

	return false
}

func (receiver *BehaviorControl) GetFollowZone() Zone {
	if receiver.target == nil {
		return NoZone
	}
	tzone := receiver.target.GetZone()
	return Zone{
		X: tzone.X + receiver.targetOffset.X,
		Y: tzone.Y + receiver.targetOffset.Y,
	}
}

func (receiver *BehaviorControl) GetDirection2Zone(zone Zone) Point {
	//todo cache center pos in zone
	center := receiver.avatar.GetCenter2()
	centerOfZone, _ := receiver.Location.CenterByIndex(zone.X, zone.Y)

	var delta float64
	var direction Point
	if delta = centerOfZone.Y - center.Y; math.Abs(delta) > 0.1 {
		if delta > 0 {
			direction.Y = 1
		} else {
			direction.Y = -1
		}
	}
	if delta = centerOfZone.X - center.X; math.Abs(delta) > 0.1 {
		if delta > 0 {
			direction.X = 1
		} else {
			direction.X = -1
		}
	}

	return direction
}

func (receiver *BehaviorControl) GetDirection2Target(target *Unit) Point {
	centerOfZone := target.GetCenter2()
	return receiver.GetDirection2Point(Point(centerOfZone))
}

func (receiver *BehaviorControl) GetDirection2Point(point Point) Point {
	//todo cache center pos in zone
	center := receiver.avatar.GetCenter2()
	centerOfZone := Center(point)

	var delta float64
	var direction Point
	if delta = centerOfZone.Y - center.Y; math.Abs(delta) > 0.5 {
		if delta > 0 {
			direction.Y = 1
		} else {
			direction.Y = -1
		}
	}
	if delta = centerOfZone.X - center.X; math.Abs(delta) > 0.5 {
		if delta > 0 {
			direction.X = 1
		} else {
			direction.X = -1
		}
	}

	return direction
}

func (receiver *BehaviorControl) Stop() (done bool) {
	if receiver.IsStop() {
		return true
	}
	receiver.commandChanel <- controller.Command{
		CType:  controller.CTYPE_MOVE,
		Pos:    controller.PosIrrelevant,
		Action: false,
	}
	return false
}

func (receiver *BehaviorControl) IsStop() bool {
	return receiver.avatar.moving
}

func (receiver *BehaviorControl) MoveToZone(zone Zone, timeLeft time.Duration) (done bool, err error) {
	//todo cache center pos in zone
	center := receiver.avatar.GetCenter2()
	avatarSpeed := receiver.avatar.Speed
	centerOfZone, _ := receiver.Location.CenterByIndex(zone.X, zone.Y)

	moveCommand := controller.Command{
		CType:  controller.CTYPE_MOVE,
		Pos:    controller.Point{},
		Action: true,
	}
	speedCommand := controller.Command{
		CType:  controller.CTYPE_SPEED_FACTOR,
		Pos:    controller.Point{1, 1},
		Action: true,
	}

	var delta, absDelta float64
	if delta = centerOfZone.Y - center.Y; math.Abs(delta) > 0.1 {
		absDelta = math.Abs(delta)
		if delta > 0 {
			moveCommand.Pos.Y = 1
		} else {
			moveCommand.Pos.Y = -1
		}
		frameSpeedY := avatarSpeed.Y * (float64(timeLeft) / float64(time.Second))
		if absDelta < frameSpeedY {
			speedCommand.Pos.Y = absDelta / frameSpeedY
		} else {
			speedCommand.Pos.Y = 1
		}
	}
	if delta = centerOfZone.X - center.X; math.Abs(delta) > 0.1 {
		absDelta = math.Abs(delta)
		if delta > 0 {
			moveCommand.Pos.X = 1
		} else {
			moveCommand.Pos.X = -1
		}
		frameSpeedX := avatarSpeed.X * (float64(timeLeft) / float64(time.Second))
		if absDelta < frameSpeedX {
			speedCommand.Pos.X = absDelta / frameSpeedX
		} else {
			speedCommand.Pos.X = 1
		}
	}

	if moveCommand.Pos.X == 0 && moveCommand.Pos.Y == 0 {
		receiver.commandChanel <- controller.Command{
			CType:  controller.CTYPE_MOVE,
			Pos:    controller.PosIrrelevant,
			Action: false,
		}
		return true, nil
	} else {
		if ok := receiver.blockedDirection[Point{moveCommand.Pos.X, 0}]; ok {
			moveCommand.Pos.X = 0
			speedCommand.Pos.X = 1
		}
		if ok := receiver.blockedDirection[Point{0, moveCommand.Pos.Y}]; ok {
			moveCommand.Pos.Y = 0
			speedCommand.Pos.Y = 1
		}
		if moveCommand.Pos.X == 0 && moveCommand.Pos.Y == 0 {
			return false, MoveBlockedError
		}
	}

	if moveCommand.Pos.Y != 0 { //only one direction at once
		moveCommand.Pos.X = 0
	}

	receiver.commandChanel <- moveCommand
	receiver.commandChanel <- speedCommand

	return false, nil
}

func (receiver *BehaviorControl) InFireRange(point Point) bool {
	/*	azone := receiver.avatar.GetZone()
		if weapSolution, ok := receiver.solutions[receiver.avatar.Gun.GetProjectile()]; !ok {
			logger.Println("no solution for weapon ", receiver.avatar.Gun.GetProjectile())
		} else {
			logger.Print(absInt(azone.X - zone.X), absInt(azone.Y - zone.Y))
			return len(weapSolution.sampleX) > absInt(azone.X - zone.X) && len(weapSolution.sampleY) > absInt(azone.Y - zone.Y)
		}*/
	return true
}

func (receiver *BehaviorControl) CanFire(point Point) bool {
	return receiver.InFireRange(point) && !receiver.avatar.Gun.IsReloading()
}

func (receiver *BehaviorControl) IsFullBlock() bool {
	if receiver.blockedDirection[left] &&
		receiver.blockedDirection[right] &&
		receiver.blockedDirection[top] &&
		receiver.blockedDirection[bottom] {
		return true
	}
	return false
}

func (receiver *BehaviorControl) CountBlockedDirection() (counter int8) {
	if receiver.blockedDirection[left] {
		counter++
	}
	if receiver.blockedDirection[right] {
		counter++
	}
	if receiver.blockedDirection[top] {
		counter++
	}
	if receiver.blockedDirection[bottom] {
		counter++
	}
	return counter
}

func (receiver *BehaviorControl) CanHit(point Point) bool {
	return true
}

func (receiver *BehaviorControl) Fire() bool {
	if controller.DEBUG_DISARM_AI {
		return true
	}
	if receiver.avatar.IsReloading() {
		return true
	}
	receiver.commandChanel <- controller.Command{
		CType:  controller.CTYPE_FIRE,
		Pos:    controller.PosIrrelevant,
		Action: true,
	}
	return false
}

func (receiver *BehaviorControl) OnIndexUpdate(tracker *Tracker) {
	if receiver.target == nil {
		return
	}
	ax, ay := receiver.avatar.GetTracker().GetIndexes()
	//tx, ty := receiver.target.GetTracker().GetIndexes()
	follow := receiver.GetFollowZone()
	receiver.pathCalculated = false
	receiver.Navigation.SchedulePath(Zone{
		X: ax,
		Y: ay,
	}, Zone{
		X: follow.X,
		Y: follow.Y,
	}, receiver)
}

func (receiver *BehaviorControl) NewPath() {
	if receiver.target == nil {
		return
	}
	ax, ay := receiver.avatar.GetTracker().GetIndexes()
	tx, ty := receiver.target.GetTracker().GetIndexes()
	receiver.pathCalculated = false
	if err := receiver.Navigation.SchedulePath(Zone{
		X: ax,
		Y: ay,
	}, Zone{
		X: tx,
		Y: ty,
	}, receiver); err != nil {
		logger.Println(err)
	}
}

func (receiver *BehaviorControl) ReceivePath(path []Zone, jobId int64) {
	for {
		if jobId <= receiver.newPathId {
			return
		}
		curr := atomic.LoadInt64(&receiver.newPathId)
		if !atomic.CompareAndSwapInt64(&receiver.newPathId, curr, jobId) {
			continue
		}
		receiver.newPath = path
	}
}

func (receiver *BehaviorControl) LookupFireSolution(solution []*FireSolutionSample, distance float64) *FireSolutionSample {
	if distance < 0 || len(solution) <= int(distance) {
		return nil
	}
	return solution[int(distance)]
}

func (receiver *BehaviorControl) forget(target *Unit) {
	for idx, targetCandidate := range receiver.availableTargets {
		if target == targetCandidate {
			receiver.availableTargets[idx] = nil
		}
	}
}

func (receiver *BehaviorControl) memorize(target *Unit) {
	receiver.availableTargets = append(receiver.availableTargets, target)
}

func (receiver *BehaviorControl) cutoffPath(path []Zone, cutoff Zone) []Zone {
	if receiver.avatar == nil || receiver.avatar.GetTracker() == nil {
		return path[0:0]
	}
	for i, z := range path {
		if z == cutoff {
			if DEBUG_AI_PATH {
				logger.Printf("cycleId: %d, objectId: %d, cutoff %d zones", CycleID, receiver.avatar.ID, i+1)
			}
			return path[i:]
		}
	}
	return path
}

func (receiver *BehaviorControl) deattach() {
	if receiver.avatar == nil {
		return
	}
	if receiver.avatar.VisionInteractions != nil {
		//deatach
	}
	if receiver.avatar.Interactions != nil {
		receiver.avatar.Interactions.Unsubscribe(receiver)
	}
}

func (receiver *BehaviorControl) attach(object *Unit) {
	if object.VisionInteractions != nil {
		//atach
	}
	if object.Interactions != nil {
		object.Interactions.Subscribe(receiver)
	}
}

func (receiver *BehaviorControl) IsNeedRecalculateSolution() bool {
	if receiver.target == nil {
		return false
	}
	if !receiver.solutionCalculated {
		return true
	}
	if !receiver.avatar.Speed.Equal(receiver.solution.aSpd, 1.0) ||
		!receiver.target.Speed.Equal(receiver.solution.tSpd, 1.0) ||
		receiver.avatar.GetProjectile() != receiver.solution.blueprint {
		return true
	}
	return false
}

func (receiver *BehaviorControl) CalculateFireSolution() error {

	receiver.solutionCalculated = false

	projectile := receiver.projectileProto[receiver.avatar.GetProjectile()]

	if projectile == nil {
		return UndefinedProjectileError
	}

	receiver.solution, _ = NewFireSolution(receiver.avatar, projectile.Copy(), receiver.target)
	receiver.solutionCalculated = true

	return nil
}

func (receiver *BehaviorControl) OnTickCollide(object collider.Collideable, collision *ump.Collision, owner *collider.Interactions) {
	if owner == receiver.avatar.Interactions && object.HasTag("obstacle") {
		body := object.GetClBody()

		if object.GetClBody().Next != nil {
			var err error
			body, err = object.GetClBody().FindExact(collision)
			if err != nil {
				logger.Println(err)
			}
		}

		tcenterX, tcenterY := body.GetCenter()
		tw, th := body.GetWH()
		acenter := receiver.avatar.GetCenter2()
		awh := receiver.avatar.GetWH2()

		direction := receiver.GetDirection2Point(Point{tcenterX, tcenterY})

		if direction.X != 0 {
			offset := acenter.Y - tcenterY
			distance := math.Abs(offset) - (awh.H/2 + th/2)
			if distance <= -collider.GRID_COORD_TOLERANCE {
				blockedDirection := Point{direction.X, 0}
				receiver.blockedDirection[blockedDirection] = true
				receiver.blockerMap[blockedDirection] = append(receiver.blockerMap[blockedDirection], object)
			}
		}
		if direction.Y != 0 {
			offset := acenter.X - tcenterX
			//log.Print(acenter, tcenterX, awh.W, tw/2, object)
			distance := math.Abs(offset) - (awh.W/2 + tw/2)
			if distance <= -collider.GRID_COORD_TOLERANCE {
				blockedDirection := Point{0, direction.Y}
				receiver.blockedDirection[blockedDirection] = true
				receiver.blockerMap[blockedDirection] = append(receiver.blockerMap[blockedDirection], object)
			}
		}
	}
}

func (receiver *BehaviorControl) OnStartCollide(object collider.Collideable, collision *ump.Collision, owner *collider.Interactions) {

}

func (receiver *BehaviorControl) OnStopCollide(object collider.Collideable, duration time.Duration, owner *collider.Interactions) {

}

func NewAIControlBuilder(collider *collider.Collider, location *Location, nav *Navigation) (*BehaviorControlBuilder, error) {
	return &BehaviorControlBuilder{
		Collider:        collider,
		Location:        location,
		Navigation:      nav,
		projectileProto: make(map[string]*Projectile),
	}, nil
}

func NewAIControl() (*BehaviorControl, error) {
	control, _ := controller.NewNoneControl()
	idle, _ := controller.NewAIControl()
	idle.Disable()
	instance := &BehaviorControl{
		Control:            control,
		idle:               idle,
		Collider:           nil,
		Location:           nil,
		Navigation:         nil,
		Behavior:           nil,
		availableTargets:   make([]*Unit, 0),
		disabled:           true,
		solutionCalculated: false,
		commandChanel:      make(chan controller.Command),
		pathLock:           sync.Mutex{},
		blockedDirection: map[Point]bool{
			right:  false,
			left:   false,
			top:    false,
			bottom: false,
		},
		blockerMap: map[Point][]collider.Collideable{
			right:  make([]collider.Collideable, 0),
			left:   make([]collider.Collideable, 0),
			top:    make([]collider.Collideable, 0),
			bottom: make([]collider.Collideable, 0),
		},
		aiCtx: context.Background(),
	}
	go func(instance *BehaviorControl, input controller.CommandChanel, output chan controller.Command) {
		for {
			select {
			case command, ok := <-input:
				if !ok {
					close(output)
					return
				}
				output <- command
				//instance.idleCommandBuffer = append(instance.idleCommandBuffer, &command)
			}
		}
	}(instance, idle.GetCommandChanel(), instance.commandChanel)
	return instance, nil
}
