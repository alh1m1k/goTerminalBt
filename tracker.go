package main

type Center struct {
	X float64
	Y float64
}

//todo refactor

type Tracker struct {
	xIndex, yIndex             int
	maxDistance                float64
	cell                       Center
	lastX, lastY, lastW, lastH float64
	Manager                    *Location
	IsNeedUpdateZone           bool
}

func (receiver *Tracker) Update(x, y, w, h float64) bool {
	if w != receiver.lastW || h != receiver.lastH {
		halfW := w / 2
		halfH := h / 2
		receiver.maxDistance = halfW*halfW + halfH*halfH
		receiver.lastW = w
		receiver.lastH = h
	}
	if x == receiver.lastX && y == receiver.lastY {
		return false
	}

	dist := getDistance(x+w/2, y+h/2, receiver.cell.X, receiver.cell.Y)
	//NearestPos check distance < half size (math.round) fix that this
	if dist > receiver.maxDistance {
		receiver.MoveTo(x, y)
		return true
	}
	return false
}

//to implement Trackable interface due simplyfy use
func (receiver *Tracker) GetTracker() *Tracker {
	return receiver
}

//to implement Trackable interface due simplyfy use
func (receiver *Tracker) GetXY() (float64, float64) {
	return receiver.lastX, receiver.lastY
}

//to implement Trackable interface due simplyfy use
func (receiver *Tracker) GetWH() (float64, float64) {
	return receiver.lastW, receiver.lastH
}

func (receiver *Tracker) MoveTo(x, y float64) error {
	pos, xi, yi, err := receiver.Manager.NearestPos(x, y)
	if err == nil {
		receiver.cell = Center{X: pos.X + receiver.lastW/2, Y: pos.Y + receiver.lastH/2}
		receiver.xIndex = xi
		receiver.yIndex = yi
		receiver.IsNeedUpdateZone = true
	} else {
		logger.Println(err)
		return err
	}
	return nil
}

func (receiver *Tracker) GetIndexes() (int, int) {
	return receiver.xIndex, receiver.yIndex
}

func (receiver *Tracker) Copy() *Tracker {
	instance := *receiver
	return &instance
}

func (receiver *Tracker) OnIndexUpdate(newCenter *Center, newX, newY int64) {

}

func NewTracker() (*Tracker, error) {
	return new(Tracker), nil
}