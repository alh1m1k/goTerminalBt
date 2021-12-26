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
	enter  time.Duration
	leave  time.Duration
	offset Zone
}

type FireSolution struct {
	blueprint        string
	prototype        *Projectile
	baseSpeed        Point
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

func (receiver *BehaviorControl) Copy() *BehaviorControl {
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

func (receiver *BehaviorControl) GetTargetZone() Zone {
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

func (receiver *BehaviorControl) MoveToZone(zone Zone, tileLeft time.Duration) (done bool) {
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
		frameSpeedY := avatarSpeed.Y * (float64(tileLeft) / float64(time.Second))
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
		frameSpeedX := avatarSpeed.X * (float64(tileLeft) / float64(time.Second))
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

func (receiver *BehaviorControl) InFireRange(zone Zone) bool {
	/*	azone := receiver.avatar.GetZone()
		if weapSolution, ok := receiver.solutions[receiver.avatar.Gun.GetProjectile()]; !ok {
			logger.Println("no solution for weapon ", receiver.avatar.Gun.GetProjectile())
		} else {
			logger.Print(absInt(azone.X - zone.X), absInt(azone.Y - zone.Y))
			return len(weapSolution.sampleX) > absInt(azone.X - zone.X) && len(weapSolution.sampleY) > absInt(azone.Y - zone.Y)
		}*/
	return true
}

func (receiver *BehaviorControl) CanHit(zone Zone) bool {
	return true
}

func (receiver *BehaviorControl) Fire() {
	if controller.DEBUG_DISARM_AI {
		return
	}
	receiver.commandChanel <- controller.Command{
		CType:  controller.CTYPE_FIRE,
		Pos:    controller.PosIrrelevant,
		Action: true,
	}
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
	if !receiver.solutionCalculated {
		return true
	}
	if !receiver.avatar.Speed.Equal(receiver.solution.baseSpeed, 1.0) ||
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
		baseSpeed: unit.Speed,
		sampleX:   make([]*FireSolutionSample, 0, 3),
		sampleY:   make([]*FireSolutionSample, 0, 3),
	}

	var (
		zone          Zone
		timeLeft, ttl time.Duration
	)

	ProjectileConfigurator(projectile, unit) //to apply speed and direction

	ttl = projectile.Ttl
	if projectile.Ttl == 0 || projectile.Ttl > time.Second*5 {
		ttl = time.Second * 5
	}
	projectile.Ttl = time.Second * 15
	//projectile.collision = collider.NewFakeCollision(1, 1 , 1, 1)
	projectile.clearTags() //todo replace with fake collision

	//sampleX
	//startPoint := Point{1,1}
	projectile.Move(0, 0)
	projectile.Direction.X = 1
	projectile.Direction.Y = 0
	projectile.Reset()
	solution.sampleX = append(solution.sampleX, &FireSolutionSample{
		enter:  0,
		leave:  0,
		offset: NoZone,
	})
	zone = NoZone
	for timeLeft = time.Duration(0); timeLeft <= ttl; timeLeft += CYCLE {
		projectile.Update(CYCLE)
		newZone := receiver.Location.IndexByPos2(projectile.GetXY2())
		if newZone != zone {
			solution.sampleX[len(solution.sampleX)-1].leave = timeLeft
			solution.sampleX = append(solution.sampleX, &FireSolutionSample{
				enter:  timeLeft,
				leave:  0,
				offset: NoZone,
			})
			zone = newZone
		}
	}
	solution.sampleX[len(solution.sampleX)-1].leave = timeLeft

	//sampleY
	projectile.Move(0, 0)
	projectile.Direction.X = 0
	projectile.Direction.Y = 1
	projectile.Reset()
	solution.sampleY = append(solution.sampleY, &FireSolutionSample{
		enter:  0,
		leave:  0,
		offset: NoZone,
	})
	zone = NoZone
	for timeLeft = time.Duration(0); timeLeft <= ttl; timeLeft += CYCLE {
		projectile.Update(CYCLE)
		newZone := receiver.Location.IndexByPos2(projectile.GetXY2())
		if newZone != zone {
			solution.sampleY[len(solution.sampleY)-1].leave = timeLeft
			solution.sampleY = append(solution.sampleY, &FireSolutionSample{
				enter:  timeLeft,
				leave:  0,
				offset: NoZone,
			})
			zone = newZone
		}
	}
	solution.sampleY[len(solution.sampleY)-1].leave = timeLeft

	receiver.applyTargetSolution(solution, unit)

	return solution, nil
}

func (receiver *BehaviorControl) applyTargetSolution(solution *FireSolution, target *Unit) {
	dTimeX := (receiver.Location.setupUnitSize.X / target.MaxSpeed.X) * float64(time.Second)
	dTimeY := (receiver.Location.setupUnitSize.Y / target.MaxSpeed.Y) * float64(time.Second)
	projectileSolution := solution
	for _, sample := range projectileSolution.sampleX {
		dYMin := math.Round(float64(sample.enter) / dTimeX)
		dYMax := math.Round(float64(sample.leave) / dTimeX)
		if dYMin == dYMax {
			sample.offset = Zone{
				X: 0,
				Y: int(dYMin),
			}
		} else {
			dYMid := math.Round((float64(sample.enter+(sample.leave-sample.enter)) / 2) / dTimeX)
			sample.offset = Zone{
				X: 0,
				Y: int(dYMid),
			}
		}
	}
	for _, sample := range projectileSolution.sampleY {
		dYMin := math.Round(float64(sample.enter) / dTimeY)
		dYMax := math.Round(float64(sample.leave) / dTimeY)
		if dYMin == dYMax {
			sample.offset = Zone{
				X: int(dYMin),
				Y: 0,
			}
		} else {
			dYMid := math.Round((float64(sample.enter+(sample.leave-sample.enter)) / 2) / dTimeY)
			sample.offset = Zone{
				X: int(dYMid),
				Y: 0,
			}
		}
	}
	if DEBUG_FIRE_SOLUTION {
		logger.Printf("fire solution %#v", projectileSolution)
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
