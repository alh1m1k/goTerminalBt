package main

import (
	"GoConsoleBT/collider"
	"sync/atomic"
	"time"
)

//try to keep this simple, obviously
type GPipeline struct {
	*Updater
	*collider.Collider
	*Visioner
	Render Renderer
	*SpawnManager
	*AnimationManager
	*EffectManager
	*Location
	*Navigation
	stage    int64
	pipe     chan int64
	ret      chan bool
	timeLeft time.Duration
}

func (receiver *GPipeline) Execute(timeLeft time.Duration) {
	receiver.timeLeft = timeLeft
	receiver.pipe <- 1
	<-receiver.ret
	receiver.stage = 0
}

func (receiver *GPipeline) doUpdate() {
	receiver.Updater.Execute(receiver.timeLeft)
	receiver.pipe <- 1
}

func (receiver *GPipeline) doAnimate() {
	receiver.AnimationManager.Execute(receiver.timeLeft)
	receiver.pipe <- 1
}

func (receiver *GPipeline) doCollect() {
	receiver.SpawnManager.Collect()
	if receiver.Updater.NeedCompact() {
		receiver.Updater.Compact()
	}
	if receiver.Render.NeedCompact() {
		receiver.Render.Compact()
	}
	if receiver.AnimationManager.NeedCompact() {
		receiver.AnimationManager.Compact()
	}
	receiver.pipe <- 1
}

func (receiver *GPipeline) doCollide() {
	receiver.Collider.Execute(receiver.timeLeft)
	receiver.pipe <- 1
}

func (receiver *GPipeline) doRender() {
	receiver.Render.Execute(receiver.timeLeft)
	receiver.pipe <- 1
}

func (receiver *GPipeline) doEffect() {
	receiver.EffectManager.Execute(receiver.timeLeft)
	receiver.pipe <- 1
}

func (receiver *GPipeline) doMap() {
	receiver.Location.Execute(receiver.timeLeft)
	receiver.pipe <- 1
}

func (receiver *GPipeline) doVision() {
	receiver.Visioner.Execute(receiver.timeLeft)
	receiver.pipe <- 1
}

func (receiver *GPipeline) doNav() {
	receiver.Navigation.Execute(receiver.timeLeft)
	receiver.pipe <- 1
}

func (receiver *GPipeline) doSpawn() {
	receiver.SpawnManager.Execute(receiver.timeLeft)
	receiver.pipe <- 1
}

func NewGPipeline() (*GPipeline, error) {
	pl := &GPipeline{
		Updater:       nil,
		Collider:      nil,
		Render:        nil,
		SpawnManager:  nil,
		EffectManager: nil,
		pipe:          make(chan int64),
		stage:         0,
		ret:           make(chan bool),
	}
	go plDispatcher(pl)

	return pl, nil
}

func plDispatcher(pl *GPipeline) {
	for {
		select {
		case inc := <-pl.pipe:
			stage := atomic.AddInt64(&pl.stage, inc)
			switch stage {
			case 1:
				go pl.doUpdate()
				go pl.doNav()
			case 3:
				go pl.doSpawn()
				go pl.doAnimate()
			case 5:
				go pl.doCollide()
				go pl.doEffect()
				go pl.doCollect()
			case 8:
				go pl.doRender()
				go pl.doVision()
				go pl.doMap()
			case 11:
				pl.ret <- true
			}
		}
	}
}
