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
			Point{
				X: 0,
				Y: 0,
			},
			Size{
				W: 267,
				H: 63,
			},
		},
	}, nil
}

type AnimationConfig struct {
	SpriteerConfig
	Duration       time.Duration `json:"duration"`
	RepeatDuration time.Duration `json:"repeatDuration"`
	Cycled         bool          `json:"cycled"`
	Blink          time.Duration `json:"blink"`
	Length         int           `json:"length"`
	Reversed       bool          `json:"reversed"`
}

type SpriteerConfig struct {
	Type          string       `json:"type"`
	Name          string       `json:"name"`
	Path          string       `json:"path"`
	Custom        CustomizeMap `json:"custom"`
	IsTransparent bool         `json:"transparent"`
	IsAbsolute    bool         `json:"absolute"`
}

type SpriteConfig struct {
	SpriteerConfig
}

type CompositionLayerConfig struct {
	SpriteerConfig
	ZIndex  int `json:"zIndex"`
	OffsetX int `json:"offsetX"`
	OffsetY int `json:"offsetY"`
}

type CompositionConfig struct {
	SpriteerConfig
}

type MotionObjectConfig struct {
	Direction     Point
	Speed         MinMax
	AccelTime     time.Duration `json:"accelTime"`
	AccelTimeFunc string        `json:"accelTimeFunc"`
}

type SizeConfig struct {
	W float64 `json:"w"`
	H float64 `json:"h"`
}
