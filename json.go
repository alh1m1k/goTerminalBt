package main

import (
	"GoConsoleBT/controller"
	"time"
)

type MinMax struct {
	Min, Max float64
}

type GameConfig struct {
	ColWidth             float64              `json:"colWidth"`
	RowHeight            float64              `json:"rowHeight"`
	LockfreePool         bool                 `json:"lockfreePool"`
	KeyBindings          []controller.KeyBind `json:"keyBindings"`
	Box                  Box                  `json:"box"`
	disableCustomization bool
}

func NewDefaultGameConfig() (*GameConfig, error) {
	return &GameConfig{
		ColWidth:     1,
		RowHeight:    1,
		LockfreePool: true,
		KeyBindings: []controller.KeyBind{
			controller.Player1DefaultKeyBinding,
			controller.Player2DefaultKeyBinding,
		},
		Box: Box{
			LT: Point{
				X: 0,
				Y: 0,
			},
			Size: Size{
				W: 100,
				H: 100,
			},
		},
	}, nil
}

type AnimationConfig struct {
	Name           string        `json:"name"`
	Path           string        `json:"path"`
	Duration       time.Duration `json:"duration"`
	RepeatDuration time.Duration `json:"repeatDuration"`
	Cycled         bool          `json:"cycled"`
	Blink          time.Duration `json:"blink"`
	Length         int           `json:"length"`
	Custom         CustomizeMap  `json:"custom"`
	Reversed       bool          `json:"reversed"`
	IsTransparent  bool          `json:"transparent"`
}

type CompositionLayerConfig struct {
	ZIndex  int `json:"zIndex"`
	OffsetX int `json:"offsetX"`
	OffsetY int `json:"offsetY"`
}

type MotionObjectConfig2 struct {
	Direction     Point
	Speed         MinMax
	AccelTime     time.Duration `json:"accelTime"`
	AccelTimeFunc string        `json:"accelTimeFunc"`
}
