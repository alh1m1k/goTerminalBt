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
	point := from.GetXY()
	size  := from.GetWH()
	gX := point.X + size.W/2
	gY := point.Y + size.H/2

	explosion := object.(*Explosion)
	expSize := explosion.GetWH()

	explosion.Owner = owner
	point.X, point.Y = gX-expSize.W/2, gY-expSize.H/2
	point.X = math.Min(math.Max(point.X, 0), float64(maxX)-expSize.W-0.5) //align to border, sux but truly need
	point.Y = math.Min(math.Max(point.Y, 0), float64(maxY)-expSize.H-0.5)
	explosion.Move(point.X, point.Y)

	explosion.GetAttr().ID = -100
	explosion.GetAttr().Team = -1
	explosion.GetAttr().TeamTag = "team--1"

	return object
}

func CollectableConfigurator(object ObjectInterface, config interface{}) ObjectInterface {
	from := config.(*Unit)
	collectable := object.(*Collectable)
	point := from.GetXY()

	collectable.Owner = from
	collectable.Move(point.X, point.Y)
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

	size := projectile.GetWH()

	centerOx += params.Direction.X * size.W / 2
	centerOy += params.Direction.Y * size.H / 2
	x = centerOx - size.W/2
	y = centerOy - size.H/2

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
