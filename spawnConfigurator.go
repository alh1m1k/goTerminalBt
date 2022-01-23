package main

import (
	direct "github.com/buger/goterm"
	"math"
	"strconv"
)

var (
	maxX, maxY = direct.Width(), direct.Height() //todo do some better
)

func DefaultConfigurator(object ObjectInterface, config interface{}) ObjectInterface {
	request := config.(*SpawnRequest)

	switch object.(type) {
	case *Wall:
		wall := object.(*Wall)
		if wall.GetAttr().Team != 0 {
			wall.removeTag(wall.GetAttr().TeamTag)
		}
		wall.GetAttr().Team = request.Team
		wall.GetAttr().TeamTag = "team-" + strconv.Itoa(int(request.Team))
		wall.addTag(wall.GetAttr().TeamTag)
	case *Unit:
		unit := object.(*Unit)
		if unit.GetAttr().Team != 0 {
			unit.removeTag(unit.GetAttr().TeamTag)
		}
		unit.GetAttr().Team = request.Team
		unit.GetAttr().TeamTag = "team-" + strconv.Itoa(int(request.Team))
		unit.addTag(unit.GetAttr().TeamTag)
	}
	return object
}

func PlayerConfigurator(object ObjectInterface, config interface{}) ObjectInterface {
	player := config.(*Player)
	object.GetAttr().Team = -1
	object.GetAttr().TeamTag = "team-" + strconv.Itoa(-1)
	object.GetAttr().Player = true
	unit := object.(*Unit)
	unit.addTag(object.GetAttr().TeamTag)
	unit.Control = player.Control
	player.Unit = unit
	return object
}

func ExplosionConfigurator(object ObjectInterface, config interface{}) ObjectInterface {
	from := config.(ObjectInterface)
	owner := from.GetOwner()
	x, y := from.GetXY()
	w, h := from.GetWH()
	gX := x + w/2
	gY := y + h/2

	explosion := object.(*Explosion)
	expW, expH := explosion.GetWH()

	explosion.Owner = owner
	x, y = gX-expW/2, gY-expH/2
	x = math.Min(math.Max(x, 0), float64(maxX)-expW-0.5) //align to border, sux but truly need
	y = math.Min(math.Max(y, 0), float64(maxY)-expH-0.5)
	explosion.Move(x, y)

	explosion.GetAttr().ID = -100
	explosion.GetAttr().Team = -1
	explosion.GetAttr().TeamTag = "team--1"

	return object
}

func CollectableConfigurator(object ObjectInterface, config interface{}) ObjectInterface {
	from := config.(*Unit)
	collectable := object.(*Collectable)
	x, y := from.GetXY()

	collectable.Owner = from
	collectable.Move(x, y)
	collectable.GetAttr().ID = -1
	collectable.GetAttr().Team = -100
	collectable.GetAttr().TeamTag = "team--100"

	return object
}

func ProjectileConfigurator(object ObjectInterface, config interface{}) ObjectInterface {
	var params FireParams
	var ok bool
	var centerOx, centerOy, x, y float64

	if params, ok = config.(FireParams); !ok {
		return nil
	}
	centerOx, centerOy = params.Position.X, params.Position.Y

	projectile := object.(*Projectile)
	object.GetAttr().Team = math.MaxInt8

	//need for proper aligment
	if params.Direction.X > 0 {
		projectile.Enter("right")
	}
	if params.Direction.X < 0 {
		projectile.Enter("left")
	}
	if params.Direction.Y < 0 {
		projectile.Enter("top")
	}
	if params.Direction.Y > 0 {
		projectile.Enter("bottom")
	}

	ow, oh := projectile.GetWH()

	centerOx += params.Direction.X * ow / 2
	centerOy += params.Direction.Y * oh / 2
	x = centerOx - ow/2
	y = centerOy - oh/2

	projectile.Move(x, y)
	//----- speed modify based at owner speed
	projectile.ApplySpeed(params.BaseSpeed)
	//-----
	projectile.Direction.X = params.Direction.X
	projectile.Direction.Y = params.Direction.Y
	projectile.Owner = params.Owner

	if projectile.GetAttr().Team != 0 {
		projectile.removeTag(projectile.GetAttr().TeamTag)
	}

	projectile.GetAttr().Team = params.Owner.GetAttr().Team
	projectile.GetAttr().TeamTag = params.Owner.GetAttr().TeamTag
	projectile.addTag(projectile.GetAttr().TeamTag)

	return projectile
}

func FanoutProjectileConfigurator(object ObjectInterface, config interface{}) ObjectInterface {
	cfg := config.(*fanoutConfig)
	owner := cfg.Owner.GetOwner().(*Unit)
	dir := cfg.Direction
	scale := cfg.SpeedScale

	projectile := object.(*Projectile)
	projectile.Direction.X = dir.X
	projectile.Direction.Y = dir.Y
	projectile.Owner = owner
	projectile.AccelTimeFunc = GetRandomTimeFunc()

	projectile.Speed.X *= scale
	projectile.Speed.Y *= scale
	projectile.MaxSpeed.X *= scale
	projectile.MaxSpeed.Y *= scale
	projectile.MinSpeed.X *= scale
	projectile.MinSpeed.Y *= scale

	if projectile.Team != 0 {
		projectile.removeTag(projectile.GetAttr().TeamTag)
	}

	projectile.GetAttr().Team = owner.Team
	projectile.GetAttr().TeamTag = owner.TeamTag
	projectile.addTag("team--1")

	return projectile
}
