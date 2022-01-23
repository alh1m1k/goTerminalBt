package main

import (
	"GoConsoleBT/collider"
	bytes2 "bytes"
	"errors"
	"log"
	"math"
	"math/rand"
	"os"
	"sync"
	"time"
)

const BORDER_SIZE = 10

var (
	ZoneSetupError     = errors.New("zone must be setup before use")
	ZoneRangeError     = errors.New("zone out of range")
	ZoneEmptyError     = errors.New("no zones left")
	ZoneCollisionError = errors.New("zone collision")

	ZoneSpawnPlaceholder = new(Tracker)
	NoPos                = Point{}
	NoCenter             = Center{}

	minimapBuf, _ = os.OpenFile("minimap.txt", os.O_CREATE|os.O_TRUNC, 644)
	minimap       = log.New(minimapBuf, "logger: ", log.Lshortfile)
)

type Trackable interface {
	GetXY() (x, y float64)
	GetWH() (w, h float64)
	GetTracker() *Tracker
}

type Location struct {
	left, right, top, bottom *collider.ClBody
	setupSize                *Size
	setupUnitSize            *Point
	zones                    [][]Trackable
	zoneX, zoneY             int
	zonesLeft                int
	zoneLock                 sync.Mutex
}

func (receiver *Location) Add(object Trackable) {
	if tracker := object.GetTracker(); tracker != nil {
		receiver.zoneLock.Lock()
		tracker.Manager = receiver
		x, y := object.GetXY()
		w, h := object.GetWH()
		tracker.Update(x, y, w, h)
		xi, yi := tracker.GetIndexes()
		if receiver.zones[yi][xi] == ZoneSpawnPlaceholder {
			receiver.zones[yi][xi] = nil
			receiver.zonesLeft++
		}
		err := receiver.putInZone(object)
		if err != nil {
			logger.Printf("error on add object to location %d, %d, %s \n", xi, yi, err)
		} else {
			receiver.zonesLeft--
		}
		receiver.zoneLock.Unlock()
	}
}

func (receiver *Location) Remove(object Trackable) {
	var tracker *Tracker
	if tracker = object.GetTracker(); tracker == nil {
		return
	}
	receiver.zoneLock.Lock()
	defer receiver.zoneLock.Unlock()
	if !tracker.IsNeedUpdateZone {
		zxi, zyi := tracker.GetIndexes()
		if receiver.zones[zyi][zxi] == object {
			receiver.zones[zyi][zxi] = nil
			tracker.Manager = nil
			receiver.zonesLeft++
			return
		} else {
			logger.Println("Position::Remove wrong index")
		}
	}
	for yi, row := range receiver.zones {
		for xi, candidate := range row {
			if object == candidate {
				receiver.zones[yi][xi] = nil
				tracker.Manager = nil
				receiver.zonesLeft++
			}
		}
	}
}

func (receiver *Location) Compact() {

}

func (receiver *Location) NeedCompact() bool {
	return false
}

func (receiver *Location) Execute(timeLeft time.Duration) {
	receiver.zoneLock.Lock()
	defer receiver.zoneLock.Unlock()
	for yi, row := range receiver.zones {
		for xi, object := range row {
			if object == nil {
				continue
			}
			if object == ZoneSpawnPlaceholder {
				receiver.zones[yi][xi] = nil
				receiver.zonesLeft++
				continue
			}
			if object.GetTracker().IsNeedUpdateZone {
				err := receiver.putInZone(object)
				if err == nil {
					receiver.zones[yi][xi] = nil
				}
			}
		}
	}
}

func (receiver *Location) Minimap(withSpawnPoint bool, applyRoutes [][]Zone) ([][]byte, error) {
	receiver.zoneLock.Lock()
	defer receiver.zoneLock.Unlock()
	minimap := make([][]byte, receiver.zoneY)

	empty := []byte(" ")
	for i, _ := range minimap {
		minimap[i] = bytes2.Repeat(empty, receiver.zoneX)
	}

	if applyRoutes != nil {
		for _, route := range applyRoutes {
			for _, zone := range route {
				minimap[zone.Y][zone.X] = byte('+')
			}
		}
	}

	for _, row := range receiver.zones {
		for _, object := range row {

			if object == nil || object.GetTracker() == nil {
				continue
			}

			x, y := object.GetTracker().GetIndexes()

			if object == ZoneSpawnPlaceholder && withSpawnPoint {
				minimap[y][x] = byte('S')
			} else {
				switch object.(type) {
				case *Unit:
					if object.(*Unit).HasTag("player") {
						minimap[y][x] = byte('P')
					} else {
						minimap[y][x] = byte('U')
					}

				case *Wall:
					minimap[y][x] = byte('W')
				default:
					minimap[y][x] = byte('?')
				}
			}
		}
	}

	return minimap, nil
}

func (receiver *Location) Mapdata() ([]*Tracker, error) {
	receiver.zoneLock.Lock()
	defer receiver.zoneLock.Unlock()
	mapdata := make([]*Tracker, 0, receiver.zoneX*receiver.zoneY-receiver.zonesLeft)

	for _, row := range receiver.zones {
		for _, object := range row {
			if object == nil || object.GetTracker() == nil {
				continue
			}

			mapdata = append(mapdata, object.GetTracker())
		}
	}

	return mapdata, nil
}

func (receiver *Location) Coordinate2Spawn(empty bool) (Point, error) {
	if receiver.zones == nil {
		return NoPos, ZoneSetupError
	}

	receiver.zoneLock.Lock()
	defer receiver.zoneLock.Unlock()

	if !empty { //generate random coord in grid, no zone taked
		xi := rand.Intn(receiver.zoneX)
		yi := rand.Intn(receiver.zoneY)
		return newPointFromZone(
			receiver.setupUnitSize.X,
			receiver.setupUnitSize.Y,
			xi,
			yi,
		), nil
	}
	if receiver.zonesLeft < 1 {
		return NoPos, ZoneEmptyError
	}
	deadline := 100
	for true {
		xi := rand.Intn(receiver.zoneX)
		yi := rand.Intn(receiver.zoneY)
		if receiver.zones[yi][xi] == nil {
			receiver.zones[yi][xi] = ZoneSpawnPlaceholder
			receiver.zonesLeft--
			return newPointFromZone(
				receiver.setupUnitSize.X,
				receiver.setupUnitSize.Y,
				xi,
				yi,
			), nil
		}
		if deadline <= 0 {
			return NoPos, ZoneEmptyError
		}
		deadline--
	}

	return NoPos, ZoneSetupError
}

func (receiver *Location) CaptureZone(zone Zone) error {
	if receiver.zones == nil {
		return ZoneSetupError
	}

	receiver.zoneLock.Lock()
	defer receiver.zoneLock.Unlock()

	if receiver.zonesLeft < 1 {
		return ZoneEmptyError
	}

	if receiver.zones[zone.Y][zone.X] == nil {
		receiver.zones[zone.Y][zone.X] = ZoneSpawnPlaceholder
		receiver.zonesLeft--
	}

	return ZoneEmptyError
}

func (receiver *Location) CapturePoint(point Point) error {
	zone := receiver.ZoneByCoordinate(point)
	return receiver.CaptureZone(zone)
}

func (receiver *Location) CoordinateByIndex(x, y int) (Point, error) {

	if x >= receiver.zoneX || y >= receiver.zoneY {
		return NoPos, ZoneRangeError
	}

	return Point{
		X: float64(x) * receiver.setupUnitSize.X,
		Y: float64(y) * receiver.setupUnitSize.Y,
	}, nil
}

func (receiver *Location) CoordinateByZone(zone Zone) (Point, error) {

	if zone.X >= receiver.zoneX || zone.Y >= receiver.zoneY {
		return NoPos, ZoneRangeError
	}

	return Point{
		X: float64(zone.X) * receiver.setupUnitSize.X,
		Y: float64(zone.Y) * receiver.setupUnitSize.Y,
	}, nil
}

func (receiver *Location) CenterByIndex(x, y int) (Center, error) {

	if x >= receiver.zoneX || y >= receiver.zoneY {
		return NoCenter, ZoneRangeError
	}

	return Center{
		X: float64(x)*receiver.setupUnitSize.X + receiver.setupUnitSize.X/2,
		Y: float64(y)*receiver.setupUnitSize.Y + receiver.setupUnitSize.Y/2,
	}, nil
}

func (receiver *Location) NearestZoneByCoordinate(point Point) Zone {
	xi, yi := int(math.Round(point.X/receiver.setupUnitSize.X)),
		int(math.Round(point.Y/receiver.setupUnitSize.Y))

	if xi < 0 {
		xi = 0
	}
	if xi >= receiver.zoneX {
		xi = receiver.zoneX - 1
	}
	if yi < 0 {
		yi = 0
	}
	if yi >= receiver.zoneY {
		yi = receiver.zoneY - 1
	}

	return Zone{
		X: xi,
		Y: yi,
	}
}

func (receiver *Location) AffectedZone(x, y float64, buffer []Point) ([]Point, error) {
	panic("not implemented")
}

func (receiver *Location) Center2Coordinate(center Center) Point {
	center.X -= receiver.setupUnitSize.X / 2
	center.Y -= receiver.setupUnitSize.Y / 2
	return Point(center)
}

func (receiver *Location) Coordinate2Center(coordinate Point) Center {
	coordinate.X += receiver.setupUnitSize.X / 2
	coordinate.Y += receiver.setupUnitSize.Y / 2
	return Center(coordinate)
}

func (receiver *Location) IndexByPos(x, y float64) (xi, yi int) {
	xi, yi = int(x/receiver.setupUnitSize.X), int(y/receiver.setupUnitSize.Y)

	if xi < 0 {
		xi = 0
	}
	if xi >= receiver.zoneX {
		xi = receiver.zoneX - 1
	}
	if yi < 0 {
		yi = 0
	}
	if yi >= receiver.zoneY {
		yi = receiver.zoneY - 1
	}

	return xi, yi
}

func (receiver *Location) ZoneByCoordinate(point Point) Zone {
	xi, yi := int(point.X/receiver.setupUnitSize.X), int(point.Y/receiver.setupUnitSize.Y)

	if xi < 0 {
		xi = 0
	}
	if xi >= receiver.zoneX {
		xi = receiver.zoneX - 1
	}
	if yi < 0 {
		yi = 0
	}
	if yi >= receiver.zoneY {
		yi = receiver.zoneY - 1
	}

	return Zone{
		X: xi,
		Y: yi,
	}
}

func (receiver *Location) Setup(pos Point, size Size) error {
	return receiver.setup(&pos, &size)
}

func (receiver *Location) SetupZones(size Point) error {
	receiver.setupUnitSize = &size
	receiver.zoneX = int(receiver.setupSize.W / size.X)
	receiver.zoneY = int(receiver.setupSize.H / size.Y)
	receiver.zones = make([][]Trackable, receiver.zoneY)
	for ri, _ := range receiver.zones {
		receiver.zones[ri] = make([]Trackable, receiver.zoneX)
	}
	receiver.zonesLeft = receiver.zoneX * receiver.zoneY
	return nil
}

func (receiver *Location) setup(pos *Point, size *Size) error {
	receiver.setupSize = size
	if receiver.left == nil {
		receiver.left = collider.NewStaticCollision(
			pos.X-BORDER_SIZE,
			pos.Y,
			BORDER_SIZE,
			size.H,
		)
	} else {
		panic("location resize not implemented")
	}
	if receiver.right == nil {
		receiver.right = collider.NewStaticCollision(
			pos.X+size.W,
			pos.Y,
			BORDER_SIZE,
			size.H,
		)
	} else {
		panic("location resize not implemented")
	}
	if receiver.top == nil {
		receiver.top = collider.NewStaticCollision(
			pos.X-BORDER_SIZE,
			pos.Y-BORDER_SIZE,
			BORDER_SIZE+size.W,
			BORDER_SIZE,
		)
	} else {
		panic("location resize not implemented")
	}
	if receiver.bottom == nil {
		receiver.bottom = collider.NewStaticCollision(
			pos.X-BORDER_SIZE,
			pos.Y+size.H,
			BORDER_SIZE+size.W,
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
	if tag == "border" {
		return true
	}
	return false
}

func NewLocation(pos Point, size Size) (*Location, error) {
	location := &Location{
		left:   nil,
		right:  nil,
		top:    nil,
		bottom: nil,
	}
	location.setup(&pos, &size)
	return location, nil
}

func (receiver *Location) putInZone(object Trackable) error {
	tracker := object.GetTracker()
	zxi, zyi := tracker.GetIndexes()
	tracker.IsNeedUpdateZone = false
	if zxi >= receiver.zoneX || zyi >= receiver.zoneY || zxi < 0 || zyi < 0 {
		return ZoneRangeError
	}
	if receiver.zones[zyi][zxi] != nil {
		if receiver.zones[zyi][zxi] == object {
			return ZoneCollisionError
		} else if receiver.zones[zyi][zxi] == ZoneSpawnPlaceholder {
			return ZoneCollisionError
		} else if receiver.zones[zyi][zxi].GetTracker().IsNeedUpdateZone {
			err := receiver.putInZone(receiver.zones[zyi][zxi])
			if err != nil {
				return err
			}
		}
		if receiver.zones[zyi][zxi] != nil {
			return ZoneCollisionError
		}
	}
	if DEBUG {
		logger.Printf("put in zone %d %d", zxi, zyi)
	}
	receiver.zones[zyi][zxi] = object
	return nil
}

func newPointFromZone(uw, uh float64, zoneX, zoneY int) Point {
	return Point{
		X: float64(zoneX) * uw,
		Y: float64(zoneY) * uh,
	}
}
