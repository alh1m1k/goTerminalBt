package main

import (
	direct "github.com/buger/goterm"
	"github.com/eiannone/keyboard"
)

const DIALOG_EVENT_COMPLETE = 500
const DIALOG_EVENT_PLAYER_SELECT = 501

var DialogCompleteEvent Event = Event{
	EType:   DIALOG_EVENT_COMPLETE,
	Payload: nil,
}

var DialogPlayerSelectEvent Event = Event{
	EType:   DIALOG_EVENT_PLAYER_SELECT,
	Payload: nil,
}

type Screener interface {
	Renderable
}

type Screen struct {
	sprite Spriteer
	size   *Point
}

func (receiver *Screen) GetXY() (x float64, y float64) {
	return float64(direct.Width()) / 2 - receiver.size.X / 2, float64(direct.Height()) / 2 - receiver.size.Y / 2
}

func (receiver *Screen) GetSprite() Spriteer {
	return receiver.sprite
}

type DialogInfo struct {
	Spriteer
	Value 	int
	Done	bool
}

type Dialog struct {
	*Screen
	keyboard  	<-chan keyboard.KeyEvent
	termination chan bool
	*State
	*ObservableObject
	CompleteEvent Event
	active		bool
	Value 		int
}

func (receiver *Dialog) ApplyState(current *StateItem) error  {
	info := current.StateInfo.(*DialogInfo)
	SwitchSprite(info.Spriteer, receiver.sprite)
	receiver.sprite = info.Spriteer
	if info.Done {
		info.Value = receiver.Value
		receiver.Trigger(receiver.CompleteEvent, receiver, info)
	} else {
		receiver.Value  = info.Value
	}
	return nil
}

func (receiver *Dialog) Deactivate()  {
	if !receiver.active {
		return
	}
	close(receiver.termination)
}

func (receiver *Dialog) Activate() error  {
	if receiver.active {
		return nil
	}
	receiver.termination = make(chan bool)
	go dialogEventDispatcher(receiver, receiver.keyboard, receiver.termination)
	receiver.active = true
	return nil
}

func NewScreen(sprite Spriteer) (*Screen, error)  {
	return &Screen{
		sprite: sprite,
	}, nil
}

func NewWinScreen() (*Screen, error)  {
	sprite, err := GetSprite("win", true, false)
	screen, _ := NewScreen(sprite)
	screen.size = &Point{200,64}
	return screen, err
}

func NewLoseScreen() (*Screen, error)  {
	sprite, err := GetSprite("lose", true, false)
	if err != nil {
		return nil, err
	}
	screen, _ := NewScreen(sprite)
	screen.size = &Point{200,64}
	return screen, err
}

func NewLogoScreen() (*Screen, error)  {
	sprite, err := GetSprite("logo", true, false)
	if err != nil {
		return nil, err
	}
	screen, _ := NewScreen(sprite)
	screen.size = &Point{200,56}
	return screen, err
}

func NewPlayerSelectDialog(keyboard <- chan keyboard.KeyEvent, chanel EventChanel) (*Dialog, error)  {
	logo, err 		:= GetSprite("logo", true, false)
	if err != nil {
		return nil, err
	}
	onePlayer, _ 	:= GetSprite("player_0", true, false)
	if err != nil {
		return nil, err
	}
	twoPlayer, _ 	:= GetSprite("player_1", true, false)
	if err != nil {
		return nil, err
	}

	offsetX := direct.Width() / 2 - 100 / 2
	offsetY := (direct.Height() / 2 - 18 / 2) + int(float64(direct.Height()) * 0.3)

	compP1, _ := NewComposition(nil)
	compP1.addFrame(logo, 0, 0, 0)
	compP1.addFrame(onePlayer, offsetX, offsetY, 1)

	compP2, _ := NewComposition(nil)
	compP2.addFrame(logo, 0, 0, 0)
	compP2.addFrame(twoPlayer, offsetX, offsetY, 1)

	state, _ := NewState(nil)

	rootInfo := &DialogInfo{
		Spriteer: compP1,
		Value:  1,
		Done: false,
	}
	state.root.StateInfo = rootInfo
	state.CreateState("/one", rootInfo)
	state.CreateState("/two", &DialogInfo{
		Spriteer: compP2,
		Value:  2,
		Done: false,
	})
	state.CreateState("/done", &DialogInfo{
		Spriteer: compP2,
		Done: true,
	})

	oo, _ := NewObservableObject(chanel, nil)

	screen := &Dialog{
		Screen: &Screen{
			sprite: compP1,
			size:   &Point{198, 56},
		},
		keyboard:         keyboard,
		State:            state,
		ObservableObject: oo,
		CompleteEvent:    DialogPlayerSelectEvent,
		Value:            1,
	}

	screen.ObservableObject.Owner 	= screen
	screen.State.Owner				= screen
	state.Enter("/one")
	state.defaultPath = "/one"

	if screen.keyboard != nil {
		screen.Activate()
	}

	return screen, nil
}

func dialogEventDispatcher(screen *Dialog, kb <- chan keyboard.KeyEvent, termination <-chan bool)  {
	for  {
		select {
		case <-termination:
			return
		case event, ok := <-kb:
			if !ok {
				return
			}
			switch event.Key {
			case keyboard.KeyArrowDown:
				screen.Enter("/two")
			case keyboard.KeyArrowUp:
				screen.Enter("/one")
			case keyboard.KeyEnter:
				screen.Enter("/done")
			}
		}
	}
}
