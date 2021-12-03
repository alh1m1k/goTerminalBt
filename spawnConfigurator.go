package main

import (
	"strconv"
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
	explosion.GetClBody().Move(gX-expW/2, gY-expH/2)

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
	collectable.GetClBody().Move(x, y)
	collectable.GetAttr().ID = -1
	collectable.GetAttr().Team = -100
	collectable.GetAttr().TeamTag = "team--100"

	return object
}

func ProjectileConfigurator(object ObjectInterface, config interface{}) ObjectInterface {

	owner := config.(*Unit)
	projectile := object.(*Projectile)
	object.GetAttr().Team = -1

	x, y := owner.GetXY()
	dir := owner.Direction
	w, h := owner.GetWH()

	if dir.X == 0 && dir.Y == 0 {
		dir.Y = -1
	}

	//need for proper aligment
	if dir.X > 0 {
		projectile.Enter("right")
	}
	if dir.X < 0 {
		projectile.Enter("left")
	}
	if dir.Y < 0 {
		projectile.Enter("top")
	}
	if dir.Y > 0 {
		projectile.Enter("bottom")
	}

	ow, oh := projectile.GetWH()

	centerX := x + w/2
	centerY := y + h/2

	centerOx := centerX + (dir.X * w / 2) + (dir.X * ow / 2)
	centerOy := centerY + (dir.Y * h / 2) + (dir.Y * oh / 2)

	x = centerOx - ow/2
	y = centerOy - oh/2

	/*	if owner.HasTag("tank") {
		x += owner.Speed.X * dir.X
		y += owner.Speed.Y * dir.Y //todo remove*/
	//	}*/

	projectile.GetClBody().Move(x, y)
	//----- speed modify based at owner speed
	projectile.Speed.X += owner.Speed.X
	projectile.Speed.Y += owner.Speed.Y
	projectile.MaxSpeed.X += owner.Speed.X
	projectile.MaxSpeed.Y += owner.Speed.Y
	projectile.MinSpeed.X += owner.Speed.X
	projectile.MinSpeed.Y += owner.Speed.Y
	//-----
	projectile.Direction.X = owner.Direction.X
	projectile.Direction.Y = owner.Direction.Y
	projectile.Owner = owner

	if projectile.GetAttr().Team != 0 {
		projectile.removeTag(projectile.GetAttr().TeamTag)
	}

	projectile.GetAttr().Team = owner.Team
	projectile.GetAttr().TeamTag = owner.TeamTag
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
