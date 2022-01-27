package main

import (
	direct "github.com/buger/goterm"
	"github.com/eiannone/keyboard"
	"math"
	"strings"
)

const DIALOG_EVENT_COMPLETE = 500
const DIALOG_EVENT_PLAYER_SELECT = 501
const DIALOG_EVENT_SETUP_SIZE = 502

var DialogCompleteEvent Event = Event{
	EType:   DIALOG_EVENT_COMPLETE,
	Payload: nil,
}

var DialogPlayerSelectEvent Event = Event{
	EType:   DIALOG_EVENT_PLAYER_SELECT,
	Payload: nil,
}

var SetupSizeEvent Event = Event{
	EType:   DIALOG_EVENT_SETUP_SIZE,
	Payload: nil,
}

var FullScreenSize = Point{
	X: math.MaxInt64 - 1,
	Y: math.MaxInt64 - 1,
}

type Screener interface {
	Renderable
}

type Screen struct {
	sprite Spriteer
	size   Point
}

func (receiver *Screen) GetXY() Point {
	if receiver.size == FullScreenSize {
		return Point{}
	} else {
		return Point{float64(direct.Width())/2 - receiver.size.X/2, float64(direct.Height())/2 - receiver.size.Y/2}
	}
}

func (receiver *Screen) GetSprite() Spriteer {
	return receiver.sprite
}

type DialogInfo struct {
	Spriteer
	Value int
	Done  bool
}

type Dialog struct {
	*Screen
	keyboard   <-chan keyboard.KeyEvent
	terminator chan bool
	*State
	*ObservableObject
	CompleteEvent Event
	active        bool
	Value         int
}

func (receiver *Dialog) ApplyState(current *StateItem) error {
	info := current.StateInfo.(*DialogInfo)
	SwitchSprite(info.Spriteer, receiver.sprite)
	receiver.sprite = info.Spriteer
	if info.Done {
		info.Value = receiver.Value
		receiver.Trigger(receiver.CompleteEvent, receiver, info)
	} else {
		receiver.Value = info.Value
	}
	return nil
}

func (receiver *Dialog) Deactivate() {
	if !receiver.active {
		return
	}
	close(receiver.terminator)
}

func (receiver *Dialog) Activate() error {
	if receiver.active {
		return nil
	}
	receiver.terminator = make(chan bool)
	go dialogEventDispatcher(receiver, receiver.keyboard, receiver.terminator)
	receiver.active = true
	return nil
}

func NewScreen(sprite Spriteer) (*Screen, error) {
	return &Screen{
		sprite: sprite,
	}, nil
}

func NewWinScreen() (*Screen, error) {
	sprite, err := GetSprite("win", true, false)
	screen, _ := NewScreen(sprite)
	screen.size = Point{X: float64(sprite.Size.W), Y: float64(sprite.Size.H)}
	return screen, err
}

func NewLoseScreen() (*Screen, error) {
	sprite, err := GetSprite("lose", true, false)
	if err != nil {
		return nil, err
	}
	screen, _ := NewScreen(sprite)
	screen.size = Point{X: float64(sprite.Size.W), Y: float64(sprite.Size.H)}
	return screen, err
}

func NewLogoScreen() (*Screen, error) {
	sprite, err := GetSprite("logo", true, false)
	if err != nil {
		return nil, err
	}
	screen, _ := NewScreen(sprite)
	wh := sprite.GetInfo().Size
	screen.size = Point{X: float64(wh.W), Y: float64(wh.H)}
	return screen, err
}

func NewSetupSizeDialog(config Box, keyboard <-chan keyboard.KeyEvent, chanel EventChanel) (*Dialog, error) {
	logoS, err := GetSprite("setup_size_s", true, false)
	logoM, err := GetSprite("setup_size_m", true, false)
	right := NewContentSprite([]byte(strings.Repeat("###\n", int(math.Round(config.H)))))
	bottom := NewContentSprite([]byte(strings.Repeat(strings.Repeat("#", int(math.Round(config.W)))+"\n", 2)))
	composition, _ := NewComposition(nil)
	composition.addFrame(logoS, 0, 0, 0)
	composition.addFrame(logoM, 0, logoS.Size.H+2, 0)
	composition.addFrame(right, int(math.Round(config.W))-3, 0, 0)
	composition.addFrame(bottom, 0, int(math.Round(config.H))-2, 0) //fixme must be - 2
	if err != nil {
		return nil, err
	}

	screen, _ := NewScreen(composition)
	screen.size = FullScreenSize
	state, _ := NewState(nil)

	rootInfo := &DialogInfo{
		Spriteer: composition,
		Value:    1,
		Done:     false,
	}
	state.root.StateInfo = rootInfo
	state.CreateState("/one", rootInfo)
	state.CreateState("/done", &DialogInfo{
		Spriteer: composition,
		Done:     true,
	})

	oo, _ := NewObservableObject(chanel, nil)

	dialog := &Dialog{
		Screen:           screen,
		keyboard:         keyboard,
		State:            state,
		ObservableObject: oo,
		CompleteEvent:    SetupSizeEvent,
		Value:            1,
	}

	dialog.ObservableObject.Owner = dialog
	dialog.State.Owner = dialog
	state.Enter("/one")
	state.defaultPath = "/one"

	if dialog.keyboard != nil {
		dialog.Activate()
	}

	return dialog, err
}

func NewPlayerSelectDialog(keyboard <-chan keyboard.KeyEvent, chanel EventChanel) (*Dialog, error) {
	logo, err := GetSprite("logo", true, false)
	if err != nil {
		return nil, err
	}
	onePlayer, _ := GetSprite("player_0", true, false)
	if err != nil {
		return nil, err
	}
	twoPlayer, _ := GetSprite("player_1", true, false)
	if err != nil {
		return nil, err
	}

	lSize := logo.GetInfo().Size
	pSize := onePlayer.GetInfo().Size
	offsetX := lSize.W/2 - pSize.W/2
	offsetY := (lSize.H/2 - pSize.H/2) + int(float64(lSize.H)*0.2)

	compP1, _ := NewComposition(nil)
	compP1.addFrame(logo, 0, 0, 0)
	compP1.addFrame(onePlayer, offsetX, offsetY, 1)

	compP2, _ := NewComposition(nil)
	compP2.addFrame(logo, 0, 0, 0)
	compP2.addFrame(twoPlayer, offsetX, offsetY, 1)

	state, _ := NewState(nil)

	rootInfo := &DialogInfo{
		Spriteer: compP1,
		Value:    1,
		Done:     false,
	}
	state.root.StateInfo = rootInfo
	state.CreateState("/one", rootInfo)
	state.CreateState("/two", &DialogInfo{
		Spriteer: compP2,
		Value:    2,
		Done:     false,
	})
	state.CreateState("/done", &DialogInfo{
		Spriteer: compP2,
		Done:     true,
	})

	oo, _ := NewObservableObject(chanel, nil)

	compSize := compP1.GetInfo().Size
	screen := &Dialog{
		Screen: &Screen{
			sprite: compP1,
			size:   Point{X: float64(compSize.W), Y: float64(compSize.H)},
		},
		keyboard:         keyboard,
		State:            state,
		ObservableObject: oo,
		CompleteEvent:    DialogPlayerSelectEvent,
		Value:            1,
	}

	screen.ObservableObject.Owner = screen
	screen.State.Owner = screen
	state.Enter("/one")
	state.defaultPath = "/one"

	if screen.keyboard != nil {
		screen.Activate()
	}

	return screen, nil
}

func dialogEventDispatcher(screen *Dialog, kb <-chan keyboard.KeyEvent, termination <-chan bool) {
	for {
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
