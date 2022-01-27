package main

import (
	"fmt"
	direct "github.com/buger/goterm"
)

type frameInfo struct {
	Spriteer
	zIndex, offsetX, offsetY int
}

//simple witchout z-index
type Composition struct {
	*Sprite
	writeProxy *Sprite
	frames     []*frameInfo
	Clip2Size  bool
}

//warn offset is absolute i.e screen offset (relative not supported :9)
func (receiver *Composition) addFrame(frame Spriteer, offsetX, offsetY, zIndex int) {
	receiver.frames = append(receiver.frames, &frameInfo{
		Spriteer: frame,
		zIndex:   zIndex,
		offsetX:  offsetX,
		offsetY:  offsetY,
	})
	//calc approx size
	wh := frame.GetInfo().Size
	receiver.Sprite.Size.W = maxInt(receiver.Sprite.Size.W, offsetX+wh.W)
	receiver.Sprite.Size.H = maxInt(receiver.Sprite.Size.H, offsetY+wh.H)
}

func (receiver *Composition) Compose() {
	receiver.Sprite.Buf.Reset()
	receiver.Sprite.Size.W, receiver.Sprite.Size.H = 0, 0
	for _, frameInfo := range receiver.frames {
		if frameInfo == nil {
			continue
		}
		if frameInfo.offsetX > 0 || frameInfo.offsetY > 0 {
			fmt.Fprint(receiver.Sprite, direct.MoveTo(frameInfo.Spriteer.String(), frameInfo.offsetX, frameInfo.offsetY)) // :(
		} else {
			fmt.Fprint(receiver.Sprite, frameInfo.Spriteer) // :(
		}
		wh := frameInfo.Spriteer.GetInfo().Size
		receiver.Sprite.Size.W = maxInt(receiver.Sprite.Size.W, wh.W+frameInfo.offsetX)
		receiver.Sprite.Size.H = maxInt(receiver.Sprite.Size.H, wh.H+frameInfo.offsetY)
	}
	if receiver.writeProxy.Buf.Len() > 0 {
		fmt.Fprint(receiver.Sprite, receiver.writeProxy)
		wh := receiver.writeProxy.GetInfo().Size
		receiver.Sprite.Size.W = maxInt(receiver.Sprite.Size.W, wh.W)
		receiver.Sprite.Size.H = maxInt(receiver.Sprite.Size.H, wh.H)
	}
}

//write to proxy, write call is analog of top z-index, transparent element of frames
func (receiver *Composition) Write(p []byte) (n int, err error) {
	n, err = receiver.writeProxy.Write(p)
	receiver.writeProxy.CalculateSize()
	return n, err
}

func (receiver *Composition) String() string {
	receiver.Compose()
	return receiver.Sprite.String()
}

func (receiver *Composition) Copy() *Composition {
	instance := *receiver
	for i := 0; i < len(receiver.frames); i++ {
		instance.frames[i] = &*receiver.frames[i]
		instance.frames[i].Spriteer = CopySprite(instance.frames[i].Spriteer)
	}
	instance.writeProxy = CopySprite(receiver.writeProxy).(*Sprite)
	instance.Sprite = CopySprite(receiver.Sprite).(*Sprite)
	return &instance
}

func NewComposition(frames []Spriteer) (*Composition, error) {
	instance := new(Composition)
	instance.writeProxy = NewSprite()
	instance.Sprite = NewSprite()
	instance.Clip2Size = false
	for i, frame := range frames {
		instance.addFrame(frame, 0, 0, i)
	}

	return instance, nil
}
