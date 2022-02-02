package main

type IndexTracker interface {
	OnIndexUpdate(tracker *Tracker)
}

//todo refactor

type Tracker struct {
	xIndex, yIndex             int
	maxDistance                float64
	cell                       Center
	lastX, lastY, lastW, lastH float64
	Manager                    *Location
	IsNeedUpdateZone           bool
	subscribers                []IndexTracker
	layer                      int
}

func (receiver *Tracker) Update(pos Point, size Size) bool {
	if receiver.Manager == nil {
		logger.Println("BUG: call Tracker::Update on unmanaged track")
		return false
	}
	if size.W != receiver.lastW || size.H != receiver.lastH {
		halfW := size.W / 2
		halfH := size.H / 2
		receiver.maxDistance = halfW*halfW + halfH*halfH
		receiver.lastW = size.W
		receiver.lastH = size.H
	}
	if pos.X == receiver.lastX && pos.Y == receiver.lastY {
		return false
	}

	if receiver.cell.X == receiver.cell.Y && receiver.cell.X == 0 {
		receiver.MoveTo(pos.X, pos.Y)
		return true
	}

	dist := getDistance(pos.X+size.W/2, pos.Y+size.H/2, receiver.cell.X, receiver.cell.Y)
	//NearestZoneByCoordinate check distance < half size (math.round) fix that this
	if dist > receiver.maxDistance {
		receiver.MoveTo(pos.X, pos.Y)
		return true
	} else {

	}
	return false
}

//to implement Trackable interface due simplyfy use
func (receiver *Tracker) GetTracker() *Tracker {
	return receiver
}

//to implement Trackable interface due simplyfy use
func (receiver *Tracker) GetXY() Point {
	return Point{receiver.lastX, receiver.lastY}
}

//to implement Trackable interface due simplyfy use
func (receiver *Tracker) GetWH() Size {
	return Size{receiver.lastW, receiver.lastH}
}

//to implement Trackable interface due simplyfy use
func (receiver *Tracker) GetLayer() int {
	return receiver.layer
}

func (receiver *Tracker) MoveTo(x, y float64) error {
	var (
		zone Zone
		pos  Point
		err  error
	)
	zone = receiver.Manager.NearestZoneByCoordinate(Point{
		X: x,
		Y: y,
	})
	pos, err = receiver.Manager.CoordinateByZone(zone)

	if err == nil {
		receiver.cell = Center{X: pos.X + receiver.lastW/2, Y: pos.Y + receiver.lastH/2}
		receiver.xIndex = zone.X
		receiver.yIndex = zone.Y
		receiver.IsNeedUpdateZone = true
		go receiver.indexUpdate()
	} else {
		logger.Println(err)
		return err
	}
	return nil
}

func (receiver *Tracker) GetIndexes() (int, int) {
	return receiver.xIndex, receiver.yIndex
}

func (receiver *Tracker) GetZone() Zone {
	return Zone{X: receiver.xIndex, Y: receiver.yIndex}
}

func (receiver *Tracker) Copy() *Tracker {
	instance := *receiver
	return &instance
}

func (receiver *Tracker) indexUpdate() {
	//todo make simpler
	for _, subscriber := range receiver.subscribers {
		if subscriber == nil {
			continue
		}
		subscriber.OnIndexUpdate(receiver)
	}
}

func (receiver *Tracker) Subscribe(subscriber IndexTracker) {
	receiver.subscribers = append(receiver.subscribers, subscriber)
}

func (receiver *Tracker) Unsubscribe(subscriber IndexTracker) {
	for index, candidate := range receiver.subscribers {
		if subscriber == candidate {
			receiver.subscribers[index] = nil
		}
	}
}

func NewTracker() (*Tracker, error) {
	return new(Tracker), nil
}
