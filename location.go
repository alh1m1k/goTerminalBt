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

const (
	BORDER_SIZE            = 10
	LOCATION_LAYER_UNIT    = 0
	LOCATION_LAYER_TERRAIN = 1
	LOCATION_LAYER_AIR     = 2
)

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
	GetXY() Point
	GetWH() Size
	GetTracker() *Tracker
	GetLayer() int
}

type Location struct {
	left, right, top, bottom *collider.ClBody
	box                      Box
	setupUnitSize            Point
	zones                    [3][][]Trackable
	zonesLeft                [3]int
	sizeZone                 Zone
	zoneLock                 sync.Mutex
}

func (receiver *Location) Add(object Trackable) {
	if tracker := object.GetTracker(); tracker != nil {
		receiver.zoneLock.Lock()
		tracker.Manager = receiver
		tracker.Update(object.GetXY(), object.GetWH())
		xi, yi := tracker.GetIndexes()
		layeri := tracker.GetLayer()
		if receiver.zones[layeri][yi][xi] == ZoneSpawnPlaceholder {
			receiver.zones[layeri][yi][xi] = nil
			receiver.zonesLeft[layeri]++
		}
		err := receiver.putInZone(object)
		if err != nil {
			logger.Printf("error on add object %d %t to location %d, %d, %s \n", object.(ObjectInterface).GetAttr().ID, receiver.zones[layeri][yi][xi], xi, yi, err)
		} else {
			receiver.zonesLeft[layeri]--
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
		layeri := tracker.GetLayer()
		if receiver.zones[layeri][zyi][zxi] == object {
			receiver.zones[layeri][zyi][zxi] = nil
			tracker.Manager = nil
			receiver.zonesLeft[layeri]++
			return
		} else {
			logger.Println("Position::Remove wrong index")
		}
	}
	for layeri, layer := range receiver.zones {
		for yi, row := range layer {
			for xi, candidate := range row {
				if object == candidate {
					receiver.zones[layeri][yi][xi] = nil
					tracker.Manager = nil
					receiver.zonesLeft[layeri]++
				}
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
	for layeri, layer := range receiver.zones {
		for yi, row := range layer {
			for xi, object := range row {
				if object == nil {
					continue
				}
				if object == ZoneSpawnPlaceholder {
					receiver.zones[layeri][yi][xi] = nil
					receiver.zonesLeft[layeri]++
					continue
				}
				if object.GetTracker().IsNeedUpdateZone {
					err := receiver.putInZone(object)
					if err == nil {
						receiver.zones[layeri][yi][xi] = nil
					}
				}
			}
		}
	}
}

func (receiver *Location) Minimap(withSpawnPoint bool, applyRoutes [][]Zone) ([][]byte, error) {
	receiver.zoneLock.Lock()
	defer receiver.zoneLock.Unlock()
	minimap := make([][]byte, receiver.sizeZone.Y)

	empty := []byte(" ")
	for i, _ := range minimap {
		minimap[i] = bytes2.Repeat(empty, receiver.sizeZone.X)
	}

	if applyRoutes != nil {
		for _, route := range applyRoutes {
			for _, zone := range route {
				minimap[zone.Y][zone.X] = byte('+')
			}
		}
	}

	for _, row := range receiver.zones[LOCATION_LAYER_UNIT] {
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

	mapdata := make([]*Tracker, 0, receiver.sizeZone.X*receiver.sizeZone.Y)
	for _, layer := range receiver.zones {
		for _, row := range layer {
			for _, object := range row {
				if object == nil || object.GetTracker() == nil {
					continue
				}
				mapdata = append(mapdata, object.GetTracker())
			}
		}
	}

	return mapdata, nil
}

func (receiver *Location) Coordinate2Spawn(empty bool, layeri int) (Point, error) {
	if receiver.zones[layeri] == nil {
		return NoPos, ZoneSetupError
	}

	receiver.zoneLock.Lock()
	defer receiver.zoneLock.Unlock()

	if !empty { //generate random coord in grid, no zone taked
		xi := rand.Intn(receiver.sizeZone.X)
		yi := rand.Intn(receiver.sizeZone.Y)
		return newPointFromZone(
			receiver.box.Point,
			receiver.setupUnitSize,
			Zone{xi, yi},
		), nil
	}
	if receiver.zonesLeft[layeri] < 1 {
		return NoPos, ZoneEmptyError
	}

	xi := rand.Intn(receiver.sizeZone.X)
	yi := rand.Intn(receiver.sizeZone.Y)
	bxi, byi := xi-1, yi
	for ; yi < receiver.sizeZone.Y; yi++ {
		for ; xi < receiver.sizeZone.X; xi++ {
			zone := Zone{xi, yi}
			if receiver.captureZone(zone, layeri) {
				return newPointFromZone(
					receiver.box.Point,
					receiver.setupUnitSize,
					zone,
				), nil
			}
		}
		xi = 0
	}
	for ; byi >= 0; byi-- {
		for ; bxi >= 0; bxi-- {
			zone := Zone{bxi, byi}
			if receiver.captureZone(zone, layeri) {
				return newPointFromZone(
					receiver.box.Point,
					receiver.setupUnitSize,
					zone,
				), nil
			}
		}
		bxi = receiver.sizeZone.X - 1
	}

	return NoPos, ZoneEmptyError
}

func (receiver *Location) CaptureZone(zone Zone, layeri int) error {
	if receiver.zones[layeri] == nil {
		return ZoneSetupError
	}

	receiver.zoneLock.Lock()
	defer receiver.zoneLock.Unlock()

	if receiver.zonesLeft[layeri] < 1 {
		return ZoneEmptyError
	}

	receiver.captureZone(zone, layeri)

	return ZoneEmptyError
}

func (receiver *Location) captureZone(zone Zone, layeri int) bool {
	if receiver.zones[layeri][zone.Y][zone.X] == nil {
		receiver.zones[layeri][zone.Y][zone.X] = ZoneSpawnPlaceholder
		receiver.zonesLeft[layeri]--
		return true
	}
	return false
}

func (receiver *Location) CapturePoint(point Point, layeri int) error {
	zone := receiver.ZoneByCoordinate(point)
	return receiver.CaptureZone(zone, layeri)
}

func (receiver *Location) CoordinateByZone(zone Zone) (Point, error) {

	if zone.X >= receiver.sizeZone.X || zone.Y >= receiver.sizeZone.Y {
		return NoPos, ZoneRangeError
	}

	return Point{
		X: receiver.box.X + (float64(zone.X) * receiver.setupUnitSize.X),
		Y: receiver.box.Y + (float64(zone.Y) * receiver.setupUnitSize.Y),
	}, nil
}

func (receiver *Location) CenterByIndex(x, y int) (Center, error) {

	if x >= receiver.sizeZone.X || y >= receiver.sizeZone.Y {
		return NoCenter, ZoneRangeError
	}

	return Center{
		X: receiver.box.X + (float64(x)*receiver.setupUnitSize.X + receiver.setupUnitSize.X/2),
		Y: receiver.box.Y + (float64(y)*receiver.setupUnitSize.Y + receiver.setupUnitSize.Y/2),
	}, nil
}

func (receiver *Location) NearestZoneByCoordinate(point Point) Zone {
	xi, yi := int(math.Round((point.X-receiver.box.X)/receiver.setupUnitSize.X)),
		int(math.Round((point.Y-receiver.box.Y)/receiver.setupUnitSize.Y))

	if xi < 0 {
		xi = 0
	}
	if xi >= receiver.sizeZone.X {
		xi = receiver.sizeZone.X - 1
	}
	if yi < 0 {
		yi = 0
	}
	if yi >= receiver.sizeZone.Y {
		yi = receiver.sizeZone.Y - 1
	}

	return Zone{
		X: xi,
		Y: yi,
	}
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

func (receiver *Location) ZoneByCoordinate(point Point) Zone {
	xi, yi := int((point.X-receiver.box.X)/receiver.setupUnitSize.X), int((point.Y-receiver.box.Y)/receiver.setupUnitSize.Y)

	if xi < 0 {
		xi = 0
	}
	if xi >= receiver.sizeZone.X {
		xi = receiver.sizeZone.X - 1
	}
	if yi < 0 {
		yi = 0
	}
	if yi >= receiver.sizeZone.Y {
		yi = receiver.sizeZone.Y - 1
	}

	return Zone{
		X: xi,
		Y: yi,
	}
}

func (receiver *Location) Setup(pos Point, size Size) error {
	return receiver.setup(Box{pos, size})
}

func (receiver *Location) SetupZones(size Point) error {
	receiver.setupUnitSize = size
	receiver.sizeZone = Zone{int(receiver.box.W / size.X), int(receiver.box.H / size.Y)}
	receiver.zones = [3][][]Trackable{
		make([][]Trackable, receiver.sizeZone.Y),
		make([][]Trackable, receiver.sizeZone.Y),
		make([][]Trackable, receiver.sizeZone.Y),
	}

	for _, layer := range receiver.zones {
		for ri, _ := range layer {
			layer[ri] = make([]Trackable, receiver.sizeZone.X)
		}
	}

	receiver.zonesLeft[0] = receiver.sizeZone.X * receiver.sizeZone.Y
	receiver.zonesLeft[1] = receiver.sizeZone.X * receiver.sizeZone.Y
	receiver.zonesLeft[2] = receiver.sizeZone.X * receiver.sizeZone.Y
	return nil
}

func (receiver *Location) setup(box Box) error {
	receiver.box = box
	if receiver.left == nil {
		receiver.left = collider.NewStaticCollision(
			box.X-BORDER_SIZE,
			box.Y,
			BORDER_SIZE,
			box.H,
		)
	} else {
		panic("location resize not implemented")
	}
	if receiver.right == nil {
		receiver.right = collider.NewStaticCollision(
			box.X+box.W,
			box.Y,
			BORDER_SIZE,
			box.H,
		)
	} else {
		panic("location resize not implemented")
	}
	if receiver.top == nil {
		receiver.top = collider.NewStaticCollision(
			box.X-BORDER_SIZE,
			box.Y-BORDER_SIZE,
			BORDER_SIZE+box.W,
			BORDER_SIZE,
		)
	} else {
		panic("location resize not implemented")
	}
	if receiver.bottom == nil {
		receiver.bottom = collider.NewStaticCollision(
			box.X-BORDER_SIZE,
			box.Y+box.H,
			BORDER_SIZE+box.W,
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
	location.setup(Box{pos, size})
	return location, nil
}

func (receiver *Location) putInZone(object Trackable) error {
	tracker := object.GetTracker()
	zxi, zyi := tracker.GetIndexes()
	tracker.IsNeedUpdateZone = false
	layeri := tracker.GetLayer()
	if zxi >= receiver.sizeZone.X || zyi >= receiver.sizeZone.Y || zxi < 0 || zyi < 0 {
		return ZoneRangeError
	}
	if receiver.zones[layeri][zyi][zxi] != nil {
		if receiver.zones[layeri][zyi][zxi] == object {
			return ZoneCollisionError
		} else if receiver.zones[layeri][zyi][zxi] == ZoneSpawnPlaceholder {
			return ZoneCollisionError
		} else if receiver.zones[layeri][zyi][zxi].GetTracker().IsNeedUpdateZone {
			err := receiver.putInZone(receiver.zones[layeri][zyi][zxi])
			if err != nil {
				return err
			}
		}
		if receiver.zones[layeri][zyi][zxi] != nil {
			return ZoneCollisionError
		}
	}
	if DEBUG {
		logger.Printf("put in zone %d %d", zxi, zyi)
	}
	receiver.zones[layeri][zyi][zxi] = object
	return nil
}

//todo make size a size
func newPointFromZone(lt Point, unit Point, zone Zone) Point {
	return Point{
		X: float64(zone.X)*unit.X + lt.X,
		Y: float64(zone.Y)*unit.Y + lt.Y,
	}
}
