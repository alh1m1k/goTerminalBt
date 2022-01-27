package main

import "math"

type Located interface {
	GetXY() Point
}

type Sized interface {
	GetWH() Size
}

type GeoSized interface {
	GetWH() GeoSize
}

type GeoPoint struct {
	X, Y int
}

type GeoSize struct {
	W, H int
}

type Point struct {
	X, Y float64
}

func (receiver Point) Equal(to Point, precision float64) bool {
	if math.Abs(receiver.X-to.X) <= precision && math.Abs(receiver.Y-to.Y) <= precision {
		return true
	}
	return false
}

func (receiver Point) Plus(to Point) Point {
	return Point{
		X: receiver.X + to.X,
		Y: receiver.Y + to.Y,
	}
}

func (receiver Point) Minus(to Point) Point {
	return Point{
		X: receiver.X - to.X,
		Y: receiver.Y - to.Y,
	}
}

func (receiver Point) Abs() Point {
	return Point{
		X: math.Abs(receiver.X),
		Y: math.Abs(receiver.Y),
	}
}

type Size struct {
	W, H float64
}

func (receiver Size) Equal(to Size, precision float64) bool {
	if math.Abs(receiver.W-to.W) <= precision && math.Abs(receiver.H-to.H) <= precision {
		return true
	}
	return false
}

func (receiver Size) Plus(to Size) Size {
	return Size{
		W: receiver.W + to.W,
		H: receiver.H + to.H,
	}
}

type Center struct {
	X float64
	Y float64
}

func (receiver Center) Equal(to Center, precision float64) bool {
	if math.Abs(receiver.X-to.X) <= precision && math.Abs(receiver.Y-to.Y) <= precision {
		return true
	}
	return false
}

func (receiver Center) Plus(to Center) Center {
	return Center{
		X: receiver.X + to.X,
		Y: receiver.Y + to.Y,
	}
}

func (receiver Center) Minus(to Center) Center {
	return Center{
		X: receiver.X - to.X,
		Y: receiver.Y - to.Y,
	}
}

func (receiver Center) Abs() Center {
	return Center{
		X: math.Abs(receiver.X),
		Y: math.Abs(receiver.Y),
	}
}

func (receiver Center) Round() Center {
	return Center{
		X: math.Round(receiver.X),
		Y: math.Round(receiver.Y),
	}
}

type Box struct {
	Point
	Size
}

type Zone struct {
	X, Y int
}

func (receiver Zone) Equal(to Zone) bool {
	return receiver == to
}

func (receiver Zone) Plus(to Zone) Zone {
	return Zone{
		X: receiver.X + to.X,
		Y: receiver.Y + to.Y,
	}
}

func (receiver Zone) Minus(to Zone) Zone {
	return Zone{
		X: receiver.X - to.X,
		Y: receiver.Y - to.Y,
	}
}

func (receiver Zone) Abs() Zone {
	return Zone{
		X: absInt(receiver.X),
		Y: absInt(receiver.Y),
	}
}
