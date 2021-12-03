package main

import (
	"math/rand"
	"time"
)

type WeatherEffect struct {
	Point             *Point
	Sprite, EndSprite Spriteer
	speed             float64
	MaxW, MaxH        int
	time.Duration
	done bool
}

func (receiver *WeatherEffect) GetXY() (float64, float64) {
	return receiver.Point.X, receiver.Point.Y
}

func (receiver *WeatherEffect) GetSprite() Spriteer {
	if receiver.done {
		return receiver.EndSprite
	}
	return receiver.Sprite
}

func (receiver *WeatherEffect) GetZIndex() int {
	return 1100
}

func (receiver *WeatherEffect) Update(timeLeft time.Duration) error {
	if receiver.done {
		if receiver.Duration <= 0 {
			receiver.Point.Y = float64(rand.Intn(receiver.MaxH))
			receiver.Point.X = float64(rand.Intn(receiver.MaxW))
			receiver.Duration = time.Duration(rand.Intn(3) * int(time.Second))
			receiver.done = false
		}
		receiver.Duration -= timeLeft
		return nil
	} else if receiver.Duration <= 0 {
		receiver.done = true
		receiver.Duration = time.Second / 3
		return nil
	}

	if receiver.Point.X <= 0 {
		receiver.Point.X = float64(receiver.MaxW)
	} else {
		receiver.Point.X = receiver.Point.X - float64(receiver.speed/float64(TIME_FACTOR))
	}

	if receiver.Point.Y >= float64(receiver.MaxH) {
		receiver.Point.Y = 0
	} else {
		receiver.Point.Y = receiver.Point.Y + float64(receiver.speed/float64(TIME_FACTOR))
	}

	receiver.Duration -= timeLeft

	return nil
}

func NewWeatherEffect(Point *Point, MaxW, MaxH int) (*WeatherEffect, error) {
	return &WeatherEffect{
		Point:     Point,
		Sprite:    ErrorSprite,
		EndSprite: nil,
		speed:     15,
		MaxW:      MaxW,
		MaxH:      MaxH,
		Duration:  time.Duration(rand.Intn(3) * int(time.Second)),
		done:      false,
	}, nil
}
