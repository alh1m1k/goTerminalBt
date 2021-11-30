package main

import (
	"GoConsoleBT/collider"
	"errors"
	"math/rand"
	"sync"
)

const BORDER_SIZE = 10

var ZoneSetupError = errors.New("zone must be setup before use")

type Location struct {
	left, right, top, bottom *collider.ClBody
	setupSize                *Point
	setupUnitSize            *Point
	zones                    []*Point
	zoneX, zoneY             int
	zoneLock                 sync.Mutex
}

func (receiver *Location) Coordinate2Spawn(empty bool) (Point, error) {
	if receiver.zones == nil {
		return Point{}, ZoneSetupError
	}

	receiver.zoneLock.Lock()
	for true {
		index := rand.Intn(cap(receiver.zones))
		if receiver.zones[index] == nil {
			receiver.zones[index] = newPointFromZone(
				receiver.setupUnitSize.X,
				receiver.setupUnitSize.Y,
				receiver.zoneX,
				index,
			)
		} else {
			if empty {
				continue
			}
		}
		receiver.zoneLock.Unlock()
		return *receiver.zones[index], nil
	}

	return Point{}, ZoneSetupError
}

func (receiver *Location) Setup(pos, size Point) error {
	return receiver.setup(&pos, &size)
}

func (receiver *Location) SetupZones(size Point) error {
	receiver.setupUnitSize = &size
	receiver.zoneX = int(receiver.setupSize.X / size.X)
	receiver.zoneY = int(receiver.setupSize.Y / size.Y)
	receiver.zones = make([]*Point, receiver.zoneX * receiver.zoneY, receiver.zoneX * receiver.zoneY)
	return nil
}

func (receiver *Location) setup(pos *Point, size *Point) error {
	receiver.setupSize = size
	if receiver.left == nil {
		receiver.left = collider.NewStaticCollision(
			pos.X - BORDER_SIZE,
			pos.Y,
			BORDER_SIZE,
			size.Y,
		)
	} else {
		panic("location resize not implemented")
	}
	if receiver.right == nil {
		receiver.right = collider.NewStaticCollision(
			pos.X + size.X,
			pos.Y,
			BORDER_SIZE,
			size.Y,
		)
	} else {
		panic("location resize not implemented")
	}
	if receiver.top == nil {
		receiver.top = collider.NewStaticCollision(
			pos.X - BORDER_SIZE,
			pos.Y - BORDER_SIZE,
			BORDER_SIZE + size.X,
			BORDER_SIZE,
		)
	} else {
		panic("location resize not implemented")
	}
	if receiver.bottom == nil {
		receiver.bottom = collider.NewStaticCollision(
			pos.X - BORDER_SIZE,
			pos.Y + size.Y,
			BORDER_SIZE + size.X,
			BORDER_SIZE,
		)
	} else {
		panic("location resize not implemented")
	}
	collider.LinkClBody(receiver.left, receiver.right)
	collider.LinkClBody(receiver.left, receiver.top)
	collider.LinkClBody(receiver.left, receiver.bottom)
	return nil
}

func (receiver *Location) GetClBody() *collider.ClBody {
	return receiver.left
}

func (receiver *Location) HasTag(tag string) bool {
	if tag == "obstacle" {
		return true
	}
	return false
}

func NewLocation(pos Point, size Point) (*Location, error) {
	location := &Location{
		left:     nil,
		right:    nil,
		top:      nil,
		bottom:   nil,
	}
	location.setup(&pos, &size)
	return location, nil
}

func newPointFromZone(uw, uh float64, zoneX, index int) *Point  {
	return &Point{
		X: float64(index % zoneX) * uw,
		Y: float64(int(index / zoneX)) * uh,
	}
}
