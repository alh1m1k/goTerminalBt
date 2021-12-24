package main

import "math"

type Located interface {
	GetXY() (x float64, y float64)
}

type Located2 interface {
	GetXY() Point
}

type Sized interface {
	GetWH() (w float64, h float64)
}

type Sized2 interface {
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

type Box struct {
	LT Point
	Size
}

type Zone struct {
	X, Y int
}
