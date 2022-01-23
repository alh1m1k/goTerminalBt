package main

import (
	"math"
	"time"
)

type FireSolutionSample struct {
	enter    time.Duration
	leave    time.Duration
	distance float64
	Offset   Center
}

type FireSolution struct {
	blueprint          string
	prototype          *Projectile
	unit               *Unit
	aSpd, tSpd         Point
	sampleX, sampleY   []*FireSolutionSample
	solutionCalculated bool
}

func (receiver *FireSolution) Recalculate(unit, target *Unit) error {
	err := receiver.calculateProjectileSolution(unit, receiver.prototype)
	if err != nil {
		return err
	}
	receiver.applyTargetSolution(target)
	receiver.normalize()
	return nil
}

func (receiver *FireSolution) Copy() *FireSolution {
	instance := *receiver
	instance.prototype = receiver.prototype.Copy()
	return &instance
}

func (receiver *FireSolution) calculateProjectileSolution(unit *Unit, projectile *Projectile) error {
	if DEBUG_FIRE_SOLUTION {
		logger.Printf("calculating solutions for %s \n", projectile.GetAttr().Blueprint)
	}

	solution := receiver

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
	ProjectileConfigurator(projectile, unit.Gun.getParams())

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
	ProjectileConfigurator(projectile, unit.Gun.getParams()) //to apply speed and direction

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
	return nil
}

func (receiver *FireSolution) applyTargetSolution(target *Unit) {
	projectileSolution := receiver
	projectileSolution.tSpd = target.Speed
	for _, sample := range projectileSolution.sampleX {
		dt := float64(sample.enter) + (float64(sample.leave-sample.enter) / 2)
		dYMid := target.MaxSpeed.Y * (dt / float64(time.Second))
		sample.Offset = Center{
			X: 0,
			Y: dYMid,
		}
		if DEBUG_FIRE_SOLUTION {
			logger.Printf("<-- fire solution sampleX[%f][%v][%v] for projectile %s unit %s zoneOffset %f -->", sample.distance, sample.enter, sample.leave, projectileSolution.blueprint, target.GetAttr().Blueprint, sample.Offset)
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
			logger.Printf("<-- fire solution sampleY[%f][%v][%v] for projectile %s unit %s zoneOffset %f -->", sample.distance, sample.enter, sample.leave, projectileSolution.blueprint, target.GetAttr().Blueprint, sample.Offset)
		}
	}
}

func (receiver *FireSolution) normalize() {
	solution := receiver
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

func NewFireSolution(unit *Unit, projectile *Projectile, target *Unit) (*FireSolution, error) {
	solution := &FireSolution{
		blueprint: projectile.GetAttr().Blueprint,
		prototype: projectile,
		unit:      unit,
		aSpd:      unit.Speed,
		sampleX:   make([]*FireSolutionSample, 0, 3),
		sampleY:   make([]*FireSolutionSample, 0, 3),
	}

	err := solution.Recalculate(unit, target)

	return solution, err
}
