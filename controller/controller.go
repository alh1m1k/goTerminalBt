package controller

import (
	"fmt"
	"github.com/eiannone/keyboard"
	"log"
	"math/rand"
	"os"
	"time"
)

const (
	CTYPE_DIRECTION = iota
	CTYPE_MOVE
	CTYPE_SPEED_FACTOR
	CTYPE_FIRE
	CTYPE_ALT_FIRE
)

var (
	buf, _ = os.OpenFile("./control.log", os.O_CREATE|os.O_TRUNC, 644)
	logger = log.New(buf, "logger: ", log.Lshortfile)
	PosIrrelevant = Point{-100, -100}
	DEBUG_DISARM_AI = false
)

var Player1DefaultKeyBinding KeyBind = KeyBind{
	Up: 	keyboard.KeyArrowUp,
	Down: 	keyboard.KeyArrowDown,
	Left: 	keyboard.KeyArrowLeft,
	Right: 	keyboard.KeyArrowRight,
	Fire: 	keyboard.KeySpace,
}

var Player2DefaultKeyBinding KeyBind = KeyBind{
	Up: 	keyboard.Key('w'),
	Down: 	keyboard.Key('s'),
	Left: 	keyboard.Key('a'),
	Right: 	keyboard.Key('d'),
	Fire: 	keyboard.KeyBackspace,
}

var KeyboardBindingPool = [2]KeyBind{
	Player1DefaultKeyBinding, Player2DefaultKeyBinding,
}

type Point struct {
	X, Y float64
}

type Command struct {
	CType  int
	Pos    Point
	Action bool
}

type Event struct {
	EType   int
	Object  interface{}
	Payload interface{}
}

type CommandChanel 	<-chan Command
type EventChanel 		chan Event

type Controller interface {
	GetCommandChanel() CommandChanel
	Enable() error
	Disable() error
}

type AwaredController interface {
	SetEventChanel(chanel EventChanel)
}

type Control struct {
	IsPlayer bool
	enabled bool
	commandChanel 	chan Command
	eventChanel 	EventChanel
	dispatcher      func(instance *Control, output chan Command, done chan bool)
	terminator      chan bool
}

func (receiver *Control) Enable() error  {
/*	if receiver.enabled != true {
		go receiver.dispatcher(receiver, receiver.commandChanel, receiver.terminator)
	}*/
	receiver.enabled = true
	return nil
}

func (receiver *Control) Disable() error {
/*	if receiver.enabled != false {
		receiver.terminator <- true
	}*/
	receiver.enabled = false
	return nil
}

func (receiver *Control) Copy() *Control {
	var control *Control
	if receiver.IsPlayer {
		control, _ = NewNoneControl()
	} else {
		copy := *receiver
		control = &copy
		control.terminator 		= make(chan bool)
		control.commandChanel 	= make(chan Command)
		go control.dispatcher(control, control.commandChanel, control.terminator)
	}
	return control
}

func (receiver *Control) GetCommandChanel() CommandChanel  {
	return receiver.commandChanel
}

func (receiver *Control) SetEventChanel(chanel EventChanel) {
	receiver.eventChanel = chanel
}

func NewPlayerControl(event <-chan keyboard.KeyEvent, keyMapping KeyBind) (*Control, error) {
	commandChanel := make(chan Command)
	instance := &Control{
		enabled:       	false,
		commandChanel: 	commandChanel,
		terminator: 	make(chan bool),
		eventChanel:   	nil,
		IsPlayer: 	   	true,
	}

	command := Command{
		Pos:  Point{},
		Action: false,
	}

	instance.dispatcher = func(instance *Control, output chan Command, done chan bool) {
		for {
			select {
			case keyEvent, ok := <-event:
				if !ok {
					close(commandChanel)
					return
				}
				if keyEvent.Key == 0 {
					keyEvent.Key = keyboard.Key(keyEvent.Rune)
				}
				if keyEvent.Key == keyboard.KeyBackspace2 {
					keyEvent.Key = keyboard.KeyBackspace //normalize backspace
				}
				switch keyEvent.Key {
				case keyMapping.Up:
					command.CType = CTYPE_MOVE
					command.Pos.Y = -1
					command.Pos.X =  0
					command.Action = true
				case keyMapping.Down:
					command.CType = CTYPE_MOVE
					command.Pos.Y =  1
					command.Pos.X =  0
					command.Action = true
				case keyMapping.Left:
					command.CType = CTYPE_MOVE
					command.Pos.X = -1
					command.Pos.Y =  0
					command.Action = true
				case keyMapping.Right:
					command.CType = CTYPE_MOVE
					command.Pos.X =  1
					command.Pos.Y =  0
					command.Action = true

				case keyMapping.Fire:
					command.CType = CTYPE_FIRE
					command.Pos 	= PosIrrelevant
					command.Action 	= true
				default:
					continue
				}
			}
			logger.Printf("send: %T, %+v \n", command, command)
			if instance.enabled {
				output <- command
			}
		}
	}
	go instance.dispatcher(instance, instance.commandChanel, instance.terminator)

	return  instance, nil
}

func NewNoneControl()(*Control, error)  {
	return &Control{
		enabled:       false,
		commandChanel: make(chan Command),
		eventChanel:   nil,
		IsPlayer: 	   false,
		terminator: make(chan bool),
		dispatcher: func(instance *Control, output chan Command, done chan bool) {
			
		},
	}, nil
}

func NewAIControl()(*Control, error)  {
	commandChanel := make(chan Command)
	instance := &Control{
		enabled:       false,
		commandChanel: commandChanel,
		terminator: make(chan bool),
		eventChanel:   nil,
		IsPlayer: 	   false,
	}

	command := Command{
		Pos:  Point{},
		Action: false,
	}

	instance.dispatcher = func(instance *Control, output chan Command, done chan bool) {
		timeEvents := time.After(time.Duration(rand.Intn(3000)) * time.Millisecond + 500)
		for {
			select {
			case _, ok := <-timeEvents:
				if !ok {
					close(commandChanel)
					return
				}
				switch rand.Intn(8) {
				case 0:
					command.CType = CTYPE_MOVE
					command.Pos.Y = -1
					command.Pos.X =  0
					command.Action = true
				case 1:
					command.CType = CTYPE_MOVE
					command.Pos.Y =  1
					command.Pos.X =  0
					command.Action = true
				case 2:
					command.CType = CTYPE_MOVE
					command.Pos.X = -1
					command.Pos.Y =  0
					command.Action = true
				case 3:
					command.CType = CTYPE_MOVE
					command.Pos.X =  1
					command.Pos.Y =  0
					command.Action = true
				}

				if rand.Intn(3) == 1 {
					if !DEBUG_DISARM_AI {
						command.CType = CTYPE_FIRE
						command.Pos = PosIrrelevant
						command.Action = true
					}
				}
			}
			if instance.enabled {
				output <- command
			}
			timeEvents = time.After(time.Duration(rand.Intn(3000)) * time.Millisecond + 500)
		}
	}
	go instance.dispatcher(instance, instance.commandChanel, instance.terminator)

	return instance, nil
}

func (c Command) String()string  {
	return fmt.Sprintf("direction %v, moving: %v, firing: %v", c.CType, c.Pos, c.Action)
}

type KeyBind struct {
	Up, Down, Left, Right, Fire keyboard.Key
}
