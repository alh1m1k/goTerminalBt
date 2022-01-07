package main

import (
	"GoConsoleBT/collider"
	"GoConsoleBT/controller"
	"errors"
	"github.com/tanema/ump"
	"math"
	"sync"
	"sync/atomic"
	"time"
)

var (
	UndefinedProjectileError = errors.New("undefined projectile")
)

type FireSolutionSample struct {
	enter    time.Duration
	leave    time.Duration
	distance float64
	Offset   Center
}

type FireSolution struct {
	blueprint        string
	prototype        *Projectile
	aSpd, tSpd       Point
	sampleX, sampleY []*FireSolutionSample
}

func (receiver *FireSolution) Copy() *FireSolution {
	instance := *receiver
	instance.prototype = receiver.prototype.Copy()
	return &instance
}

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
	targetOffset                 Zone
	availableTargets             []*Unit
	lastPath, newPath            []Zone
	newPathId                    int64
	disabled, solutionCalculated bool
	solution                     *FireSolution
	projectileProto              map[string]*Projectile
	commandChanel                chan controller.Command
	pathLock                     sync.Mutex
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
	receiver.target = object
	receiver.Next(ChosePatternBehavior)
}

func (receiver *BehaviorControl) UnSee(object *Unit) {
	receiver.Next(IdleBehavior)
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
	if path := receiver.newPath; path != nil {
		if len(receiver.lastPath) > 0 {
			receiver.lastPath = receiver.cutoffPath(path, receiver.lastPath[0])
		} else {
			receiver.lastPath = path
		}
		receiver.newPath = nil
	}
	if behavior := receiver.nextBehavior; behavior != nil {
		receiver.nextBehavior = nil
		receiver.next(behavior)
	}
	if receiver.Behavior != nil {
		receiver.Behavior.Update(receiver, timeLeft)
	}
	return nil
}

//todo return to
func (receiver *BehaviorControl) Next(behavior *Behavior) {
	if receiver.Behavior != nil && &receiver.Behavior.Update == &NoUpdate {
		panic("atatat")
	}
	receiver.nextBehavior = behavior
}

func (receiver *BehaviorControl) Copy() controller.Controller {
	instance, _ := receiver.builder.Build()
	return instance
}

func (receiver *BehaviorControl) next(behavior *Behavior) {
	if behavior == nil || receiver.Behavior == behavior {
		return
	}
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
		CType:  controller.CTYPE_DIRECTION,
		Pos:    controller.Point(direction),
		Action: true,
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
		CType:  controller.CTYPE_DIRECTION,
		Pos:    controller.Point(direction),
		Action: true,
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

func (receiver *BehaviorControl) MoveToZone(zone Zone, timeLeft time.Duration) (done bool) {
	//todo cache center pos in zone
	center := receiver.avatar.GetCenter2()
	avatarSpeed := receiver.avatar.Speed
	centerOfZone, _ := receiver.Location.CenterByIndex(zone.X, zone.Y)

	moveCommand := controller.Command{
		CType:  controller.CTYPE_DIRECTION,
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
	} else if delta = centerOfZone.X - center.X; math.Abs(delta) > 0.1 {
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
	} else {
		receiver.commandChanel <- controller.Command{
			CType:  controller.CTYPE_MOVE,
			Pos:    controller.PosIrrelevant,
			Action: false,
		}
		return true
	}

	receiver.commandChanel <- moveCommand
	receiver.commandChanel <- speedCommand

	return false
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
	return receiver.InFireRange(point) && !receiver.avatar.Gun.IsReload()
}

func (receiver *BehaviorControl) CanHit(point Point) bool {
	return true
}

func (receiver *BehaviorControl) Fire() bool {
	if controller.DEBUG_DISARM_AI {
		return true
	}
	if receiver.avatar.IsReload() {
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
	tx, ty := receiver.target.GetTracker().GetIndexes()
	/*	receiver.pathLock.Lock()
		defer receiver.pathLock.Unlock()
		if len(receiver.lastPath) > 15 { //for long run path
			lastZone := receiver.lastPath[len(receiver.lastPath) - 1]
			if tx - lastZone.X >= -1 && tx - lastZone.X <= 1 && ty - lastZone.Y >= -1 && ty - lastZone.Y <= 1 {
				logger.Printf("fast zone update %d, %d", tx, ty)
				receiver.lastPath = append(receiver.lastPath, Zone{X: tx, Y: ty})
				return
			}
		} else {
			logger.Printf("target change it's zone new is %d, %d", tx, ty)
		}*/
	receiver.Navigation.SchedulePath(Zone{
		X: ax,
		Y: ay,
	}, Zone{
		X: tx,
		Y: ty,
	}, receiver)
}

func (receiver *BehaviorControl) NewPath() {
	if receiver.target == nil {
		return
	}
	ax, ay := receiver.avatar.GetTracker().GetIndexes()
	tx, ty := receiver.target.GetTracker().GetIndexes()
	receiver.Navigation.SchedulePath(Zone{
		X: ax,
		Y: ay,
	}, Zone{
		X: tx,
		Y: ty,
	}, receiver)
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

	receiver.solution, _ = receiver.calculateFireSolution(receiver.avatar, projectile.Copy())
	receiver.applyTargetSolution(receiver.solution, receiver.target)
	receiver.normalizeTargetSolution(receiver.solution)
	receiver.solutionCalculated = true

	return nil
}

func (receiver *BehaviorControl) calculateFireSolution(unit *Unit, projectile *Projectile) (*FireSolution, error) {
	if DEBUG_FIRE_SOLUTION {
		logger.Printf("calculating solutions for %s \n", projectile.GetAttr().Blueprint)
	}
	solution := &FireSolution{
		blueprint: projectile.GetAttr().Blueprint,
		prototype: projectile,
		aSpd:      unit.Speed,
		sampleX:   make([]*FireSolutionSample, 0, 3),
		sampleY:   make([]*FireSolutionSample, 0, 3),
	}

	var (
		point         Point
		timeLeft, ttl time.Duration
	)

	ttl = projectile.Ttl
	if projectile.Ttl == 0 || projectile.Ttl > time.Second*5 {
		ttl = time.Second * 5
	}
	projectile.Ttl = time.Second * 15
	//projectile.collision = collider.NewFakeCollision(1, 1 , 1, 1)
	projectile.clearTags() //todo replace with fake collision

	projectile.Reset()
	ProjectileConfigurator(projectile, unit) //to apply speed and direction

	//sampleX
	projectile.Move(0, 0)
	projectile.Direction.X = 1
	projectile.Direction.Y = 0

	solution.sampleX = append(solution.sampleX, &FireSolutionSample{
		enter:    0,
		leave:    0,
		distance: 0.0,
	})
	point = Point{} //0:0
	for timeLeft = CYCLE / 4; timeLeft <= ttl; timeLeft += CYCLE / 4 {
		projectile.Update(CYCLE / 4)
		newPoint := projectile.GetXY2()
		if math.Round(newPoint.X) != math.Round(point.X) {
			solution.sampleX[len(solution.sampleX)-1].leave = timeLeft
			solution.sampleX = append(solution.sampleX, &FireSolutionSample{
				enter:    timeLeft,
				leave:    0,
				distance: math.Round(newPoint.X),
			})
			point = newPoint
		}
	}
	solution.sampleX[len(solution.sampleX)-1].leave = timeLeft

	projectile.Reset()
	ProjectileConfigurator(projectile, unit) //to apply speed and direction

	//sampleY
	projectile.Move(0, 0)
	projectile.Direction.X = 0
	projectile.Direction.Y = 1
	solution.sampleY = append(solution.sampleY, &FireSolutionSample{
		enter:    0,
		leave:    0,
		distance: 0.0,
	})
	point = Point{} //0:0
	for timeLeft = CYCLE / 4; timeLeft <= ttl; timeLeft += CYCLE / 4 {
		projectile.Update(CYCLE / 4)
		newPoint := projectile.GetXY2()
		if math.Round(newPoint.Y) != math.Round(point.Y) {
			solution.sampleY[len(solution.sampleY)-1].leave = timeLeft
			solution.sampleY = append(solution.sampleY, &FireSolutionSample{
				enter:    timeLeft,
				leave:    0,
				distance: math.Round(newPoint.Y),
			})
			point = newPoint
		}
	}
	solution.sampleY[len(solution.sampleY)-1].leave = timeLeft
	return solution, nil
}

func (receiver *BehaviorControl) applyTargetSolution(solution *FireSolution, target *Unit) {
	projectileSolution := solution
	projectileSolution.tSpd = target.Speed
	for _, sample := range projectileSolution.sampleX {
		dt := float64(sample.enter) + (float64(sample.leave-sample.enter) / 2)
		dYMid := target.MaxSpeed.Y * (dt / float64(time.Second))
		sample.Offset = Center{
			X: 0,
			Y: dYMid,
		}
		if DEBUG_FIRE_SOLUTION {
			logger.Printf("<-- fire solution sampleX[%f][%v][%v] for projectile %s unit %s zoneOffset %f -->", sample.distance, sample.enter, sample.leave, solution.blueprint, target.GetAttr().Blueprint, sample.Offset)
		}
	}
	for _, sample := range projectileSolution.sampleY {
		dt := float64(sample.enter) + (float64(sample.leave-sample.enter) / 2)
		dYMid := target.MaxSpeed.X * (dt / float64(time.Second))
		sample.Offset = Center{
			X: dYMid,
			Y: 0,
		}
		if DEBUG_FIRE_SOLUTION {
			logger.Printf("<-- fire solution sampleY[%f][%v][%v] for projectile %s unit %s zoneOffset %f -->", sample.distance, sample.enter, sample.leave, solution.blueprint, target.GetAttr().Blueprint, sample.Offset)
		}
	}
}

func (receiver *BehaviorControl) normalizeTargetSolution(solution *FireSolution) {
	newXLen := int(solution.sampleX[len(solution.sampleX)-1].distance) + 1
	newYLen := int(solution.sampleY[len(solution.sampleY)-1].distance) + 1
	if newXLen != len(solution.sampleX) {
		logger.Println("normalize sampleX solution")
		newSampleX := make([]*FireSolutionSample, newXLen, newXLen)
		for _, sample := range solution.sampleX {
			newSampleX[int(sample.distance)] = sample
		}
		curr := solution.sampleX[0]
		for index, sample := range newSampleX {
			if sample == nil {
				newSampleX[index] = curr
			} else {
				curr = sample
			}
		}
		solution.sampleX = newSampleX
	}
	if newYLen != len(solution.sampleY) {
		logger.Println("normalize sampleY solution")
		newSampleY := make([]*FireSolutionSample, newYLen, newYLen)
		for _, sample := range solution.sampleY {
			newSampleY[int(sample.distance)] = sample
		}
		curr := solution.sampleY[0]
		for index, sample := range newSampleY {
			if sample == nil {
				newSampleY[index] = curr
			} else {
				curr = sample
			}
		}
		solution.sampleY = newSampleY
	}
}

func (receiver *BehaviorControl) OnTickCollide(object collider.Collideable, collision *ump.Collision, owner *collider.Interactions) {

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
	}
	go func(input controller.CommandChanel, output chan controller.Command) {
		for {
			select {
			case command, ok := <-input:
				if !ok {
					close(output)
					return
				}
				output <- command
			}
		}
	}(idle.GetCommandChanel(), instance.commandChanel)
	return instance, nil
}
