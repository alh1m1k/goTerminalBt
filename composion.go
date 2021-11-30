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
	frames []*frameInfo
}

//warn offset is absolute i.e screen offset (relstive not supported :9)
func (receiver *Composition) addFrame(frame Spriteer, offsetX, offsetY, zIndex int)  {
	receiver.frames = append(receiver.frames, &frameInfo{
		Spriteer:    frame,
		zIndex:      zIndex,
		offsetX: 	 offsetX,
		offsetY: 	 offsetY,
	})
}

func (receiver *Composition) Compose()  {
	receiver.Sprite.Buf.Reset()
	for _, frameInfo := range receiver.frames {
		if frameInfo.offsetX > 0 || frameInfo.offsetY > 0 {
			fmt.Fprint(receiver.Sprite, direct.MoveTo(frameInfo.Spriteer.String(), frameInfo.offsetX, frameInfo.offsetY))  // :(
		} else {
			fmt.Fprint(receiver.Sprite, frameInfo.Spriteer)  // :(
		}
	}
	fmt.Fprint(receiver.Sprite, receiver.writeProxy)
}

//write to proxy, write call is analog of top z-index, transparent element of frames
func (receiver *Composition) Write(p []byte) (n int, err error)   {
	return receiver.writeProxy.Write(p)
}

func (receiver *Composition) String() string   {
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
	instance.Sprite 	= CopySprite(receiver.Sprite).(*Sprite)
	return &instance
}

func NewComposition(frames []Spriteer) (*Composition, error)  {
	instance := new(Composition)
	instance.writeProxy = NewSprite()
	instance.Sprite 	= NewSprite()

	for i, frame := range frames {
		instance.addFrame(frame, 0, 0, i)
	}

	return instance, nil
}
