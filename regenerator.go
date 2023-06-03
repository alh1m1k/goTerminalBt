package main

import (
	"math"
	"time"
)

var NoRegeneration = &Regenerator{}

type Regenerator struct {
	Regeneration           float64
	RegeneratorAccumulator float64
}

func (receiver *Regenerator) Update(timeLeft time.Duration) error {
	if receiver.Regeneration > 0 {
		receiver.RegeneratorAccumulator += float64(timeLeft) * (receiver.Regeneration / float64(time.Second))
		logger.Println(float64(timeLeft) * (receiver.Regeneration / float64(time.Second)))
	}
	return nil
}

func (receiver *Regenerator) GetAccumulatedRaw() float64 {
	var r float64
	r, receiver.RegeneratorAccumulator = receiver.RegeneratorAccumulator, 0
	return r
}

func (receiver *Regenerator) GetAccumulated() int {
	var r float64
	if receiver.RegeneratorAccumulator >= 1.0 {
		r = math.Ceil(receiver.RegeneratorAccumulator)
		receiver.RegeneratorAccumulator -= r
		return int(r)
	}
	return 0
}

func (receiver *Regenerator) Reset() {
	receiver.RegeneratorAccumulator = 0.0
}

func NewRegenerator(regenPerSecond float64) (*Regenerator, error) {
	return &Regenerator{
		Regeneration:           regenPerSecond,
		RegeneratorAccumulator: 0,
	}, nil
}
