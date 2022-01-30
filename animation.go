package main

import (
	"errors"
	"fmt"
	"math"
	"strconv"
	"time"
)

var animations map[string]*Animation = make(map[string]*Animation, 25)
var BlinkSprite = NewSprite() //empty sprite

var FrameTypeCombinationError = errors.New("all frame must be same type")
var UndefinedFrameCountError = errors.New("frame count must be predefined and more than 0")
var MismatchFrameCountError = errors.New("frame count param mismatch actual frame count")
var AnimationExistError = errors.New("animation exist in storage")
var Animation404Error = errors.New("animation does not exist in storage")
var AnimationCustomizationError = errors.New("animation customization error")
var AnimationEmptyCustomizationError = errors.New("empty sprite customization list")

var ErrorAnimation, _ = NewErrorAnimation()

type Animation struct {
	Spriteer
	Manager                                   *AnimationManager
	Duration, RepeatDuration                  time.Duration
	keyFrames                                 []Spriteer
	TimeFunction                              timeFunction
	Cycled, Reversed, container, collection   bool
	BlinkRate, index, blinkIndex, repeatIndex time.Duration
}

func (receiver *Animation) Update(timeLeft time.Duration) (done bool, error error) {
	if receiver.collection { //todo split struct
		return receiver.updateCollection(timeLeft) //collection of sprites
	} else {
		return receiver.updateContainer(timeLeft) //container for animation
	}
}

func (receiver *Animation) updateCollection(timeLeft time.Duration) (done bool, error error) {
	var (
		index  int64
		offset float64
	)

	offset = math.Min(receiver.TimeFunction(float64(receiver.index)/float64(receiver.Duration)), 1)
	if receiver.Reversed {
		fLen := float64(len(receiver.keyFrames) - 1)
		index = int64(fLen - offset*fLen)
	} else {
		index = int64(math.Round(offset * float64(len(receiver.keyFrames)-1)))
	}

	if receiver.BlinkRate > 0 {
		if receiver.blinkIndex > receiver.BlinkRate {
			if receiver.Spriteer == BlinkSprite {
				receiver.Spriteer = receiver.keyFrames[index]
			} else {
				receiver.Spriteer = BlinkSprite
			}
			receiver.blinkIndex = receiver.blinkIndex % receiver.BlinkRate
		}
		receiver.blinkIndex += timeLeft
	} else {
		receiver.Spriteer = receiver.keyFrames[index]
	}

	if receiver.index >= receiver.Duration {
		if receiver.Cycled {
			receiver.index = 0
		}
		if receiver.RepeatDuration < 0 {
			return true, nil
		}
	}

	if receiver.RepeatDuration > 0 {
		if receiver.repeatIndex > receiver.RepeatDuration {
			receiver.repeatIndex = 0
			return true, nil
		} else {
			receiver.repeatIndex += timeLeft
		}
	}

	receiver.index += timeLeft

	return false, nil
}

func (receiver *Animation) updateContainer(timeLeft time.Duration) (done bool, error error) {
	var animation *Animation

	if receiver.Reversed {
		index := len(receiver.keyFrames) - int(receiver.index) - 1
		animation = receiver.keyFrames[index].(*Animation)
	} else {
		animation = receiver.keyFrames[receiver.index].(*Animation)
	}

	done, error = animation.Update(timeLeft)
	receiver.Spriteer = animation.Spriteer

	if error != nil {
		return false, error
	}

	if int64(receiver.index) >= int64(len(receiver.keyFrames))-1 {
		if receiver.Cycled {
			receiver.index = 0
		}
		return true, nil
	}

	if done {
		receiver.index++
	}

	return false, nil
}

func (receiver *Animation) AddFrame(frame Spriteer) error {
	switch frame.(type) {
	case *Animation:
		if receiver.collection {
			return FrameTypeCombinationError
		}
		receiver.keyFrames = append(receiver.keyFrames, frame)
		receiver.container = true
	default:
		if receiver.container {
			return FrameTypeCombinationError
		}
		receiver.keyFrames = append(receiver.keyFrames, frame)
		receiver.collection = true
	}
	if receiver.Spriteer == nil {
		receiver.Spriteer = receiver.keyFrames[0]
	}
	return nil
}

func (receiver *Animation) Reset() {
	receiver.index = 0
	receiver.blinkIndex = 0
	receiver.repeatIndex = 0
	if receiver.collection == true {
		if receiver.Duration == 0 {
			logger.Println("animation collection mode with zero duration warning")
		}
	}
	if receiver.container == true {
		for _, frame := range receiver.keyFrames {
			if animation, ok := frame.(*Animation); ok {
				animation.Reset()
			} else {
				logger.Printf("non animation object in container")
			}
		}
	}
	if len(receiver.keyFrames) > 0 {
		if receiver.Reversed {
			receiver.Spriteer = receiver.keyFrames[len(receiver.keyFrames)-1]
		} else {
			receiver.Spriteer = receiver.keyFrames[0]
		}
	}
}

func (receiver *Animation) Copy() *Animation {
	instance := *receiver
	instance.keyFrames = make([]Spriteer, len(instance.keyFrames), cap(instance.keyFrames))
	for i := 0; i < len(receiver.keyFrames); i++ {
		instance.keyFrames[i] = CopySprite(receiver.keyFrames[i])
	}
	instance.Spriteer = CopySprite(receiver.Spriteer)
	return &instance
}

func NewAnimation(sprites []Spriteer) (*Animation, error) {
	anim := Animation{
		Spriteer:       nil,
		Duration:       0,
		keyFrames:      nil,
		TimeFunction:   LinearTimeFunction,
		Cycled:         true,
		Reversed:       false,
		container:      false,
		collection:     false,
		index:          0,
		BlinkRate:      -1,
		blinkIndex:     0,
		RepeatDuration: -1,
	}

	if sprites != nil {
		for _, sprite := range sprites {
			err := anim.AddFrame(sprite)
			if err != nil {
				return nil, err
			}
		}
	}

	return &anim, nil
}

func NewErrorAnimation() (*Animation, error) {
	anim := Animation{
		Spriteer:       nil,
		Duration:       1,
		keyFrames:      nil,
		TimeFunction:   LinearTimeFunction,
		Cycled:         true,
		Reversed:       false,
		container:      false,
		collection:     false,
		index:          0,
		BlinkRate:      -1,
		blinkIndex:     0,
		RepeatDuration: -1,
	}

	err := anim.AddFrame(ErrorSprite)

	return &anim, err
}

//return new animation every call
func GetAnimation(id string, length int, load bool, processTransparent bool) (*Animation, error) {
	if anim, ok := animations[id]; ok {
		if len(anim.keyFrames) != length {
			return nil, MismatchFrameCountError
		}
		return anim.Copy(), nil
	}

	if !load {
		return nil, Animation404Error
	}

	if length <= 0 {
		return nil, UndefinedFrameCountError
	}

	anim, err := NewAnimation(nil)
	if err != nil {
		return nil, err
	}
	for i := 0; i < length; i++ {
		sprite, err := GetSprite(id+"_"+strconv.Itoa(i), true, processTransparent)
		if err != nil {
			return nil, err
		}
		err = anim.AddFrame(sprite)
		if err != nil {
			return nil, err
		}
	}
	animations[id] = anim
	return anim.Copy(), nil
}

func GetAnimation2(id string) (*Animation, error) {
	if anim, ok := animations[id]; ok {
		return anim.Copy(), nil
	}
	return ErrorAnimation, Animation404Error
}

func LoadAnimation2(path string, length int, processTransparent bool) (*Animation, error) {
	var sprite *Sprite
	var err error

	anim, err := NewAnimation(nil)
	if err != nil {
		return ErrorAnimation, err
	}
	for i := 0; i < length; i++ {
		spriteId := path + "_" + strconv.Itoa(i)
		sprite, err = GetSprite2(spriteId)
		if err != nil {
			path := path + "_" + strconv.Itoa(i)
			sprite, err = LoadSprite2(path, processTransparent)
			if err == nil {
				AddSprite(path, sprite)
			}
		}
		if err != nil {
			return ErrorAnimation, err
		}
		err = anim.AddFrame(sprite)
		if err != nil {
			return ErrorAnimation, err
		}
	}
	return anim, nil
}

func AddAnimation(id string, anim *Animation) error {
	if _, ok := animations[id]; ok {
		return fmt.Errorf("%s, %w", id, AnimationExistError)
	}
	animations[id] = anim
	return nil
}

func CustomizeAnimation(animation *Animation, name string, custom CustomizeMap) (*Animation, error) {
	if len(custom) == 0 {
		return ErrorAnimation, AnimationEmptyCustomizationError
	}

	newAnimation := animation.Copy() //?
	newAnimation.keyFrames = animation.keyFrames[0:0]
	newAnimation.Spriteer = nil
	for i, frame := range animation.keyFrames {
		if s, ok := frame.(*Sprite); ok {
			if frameCustom, err := GetSprite2(customizedSpriteName(name+strconv.Itoa(i), custom)); err != nil {
				frameCustom, err = CustomizeSprite(s, custom)
				if err == nil {
					if err = AddSprite(customizedSpriteName(name+strconv.Itoa(i), custom), frameCustom); err == nil {
						newAnimation.AddFrame(CopySprite(frameCustom))
					} else {
						return ErrorAnimation, AnimationExistError
					}
				} else {
					return ErrorAnimation, AnimationCustomizationError
				}
			} else {
				newAnimation.AddFrame(frameCustom)
			}
		} else {
			newAnimation = ErrorAnimation
			return newAnimation, fmt.Errorf("%s: %w %t", name, AnimationCustomizationError, custom)
		}
	}

	return newAnimation, nil
}
