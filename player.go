package main

import (
	"GoConsoleBT/controller"
	"github.com/eiannone/keyboard"
	"sync/atomic"
)

type Player struct {
	*controller.Control
	Keyboard <-chan keyboard.KeyEvent
	Unit     *Unit
	*CustomizeMap
	Name      string
	Blueprint string
	Score     int64
	Retry     int32
}

func (receiver *Player) IncScore(byValue int64) int64 {
	return atomic.AddInt64(&receiver.Score, byValue)
}

func (receiver *Player) DecrRetry(byValue int32) int32 {
	return atomic.AddInt32(&receiver.Retry, byValue*-1)
}

func NewPlayer(name string, control *controller.Control) (*Player, error) {
	return &Player{
		Control:      control,
		Unit:         nil,
		CustomizeMap: nil,
		Name:         name,
		Score:        0,
		Retry:        3, //good default :)
	}, nil
}
