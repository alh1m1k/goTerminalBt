package controller

import (
	"fmt"
	"github.com/eiannone/keyboard"
	"log"
	"math/rand"
	"os"
	"time"
)

const DEBUG_DISARM_AI = false

var (
	buf, _ = os.OpenFile("control.log", os.O_CREATE|os.O_TRUNC, 644)
	logger = log.New(buf, "logger: ", log.Lshortfile)
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
	Direction Point
	Move      bool
	Fire      bool
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
}

type AwaredController interface {
	SetEventChanel(chanel EventChanel)
}

type Control struct {
	IsPlayer bool
	enabled bool
	commandChanel 	CommandChanel
	eventChanel 	EventChanel
}

func (receiver Control) Enable() error  {
	receiver.enabled = true
	return nil
}

func (receiver Control) Disable() error {
	receiver.enabled = false
	return nil
}

func (receiver *Control) GetCommandChanel() CommandChanel  {
	return receiver.commandChanel
}

func (receiver *Control) SetEventChanel(chanel EventChanel) {
	receiver.eventChanel = chanel
}

func NewPlayerControl(event <-chan keyboard.KeyEvent, keyMapping KeyBind) (*Control, error) {
	command := Command{
		Direction: Point{X:0, Y:0},
		Move:      false,
		Fire:      false,
	}
	commandChanel := make(chan Command)
	go func(event <-chan keyboard.KeyEvent, keyMapping KeyBind, output chan Command) {
		for {
			select {
			case keyEvent, ok := <-event:
				if !ok {
					close(commandChanel)
					return
				}
				command.Fire = false
				if keyEvent.Key == 0 {
					keyEvent.Key = keyboard.Key(keyEvent.Rune)
				}
				switch keyEvent.Key {
				case keyMapping.Up:
					command.Direction.Y = -1
					command.Direction.X =  0
					command.Move = true
				case keyMapping.Down:
					command.Direction.Y =  1
					command.Direction.X =  0
					command.Move = true
				case keyMapping.Left:
					command.Direction.X = -1
					command.Direction.Y =  0
					command.Move = true
				case keyMapping.Right:
					command.Direction.X =  1
					command.Direction.Y =  0
					command.Move = true
				case keyMapping.Fire:
					command.Fire = true
				}
			}
			logger.Printf("send: %T, %+v \n", command, command)
			commandChanel <- command
		}
	}(event, keyMapping, commandChanel)


	return &Control{
		enabled:       false,
		commandChanel: commandChanel,
		eventChanel:   nil,
		IsPlayer: 	   true,
	}, nil
}

func NewNoneControl()(*Control, error)  {
	return &Control{
		enabled:       false,
		commandChanel: make(CommandChanel),
		eventChanel:   nil,
		IsPlayer: 	   false,
	}, nil
}

func NewAIControl()(*Control, error)  {
	command := Command{
		Direction: Point{X:0, Y:0},
		Move:      false,
		Fire:      false,
	}
	commandChanel := make(chan Command)
	go func(output chan Command) {
		timeEvents := time.After(time.Duration(rand.Intn(3000)) * time.Millisecond + 500)
		for {
			select {
			case _, ok := <-timeEvents:
				if !ok {
					close(commandChanel)
					return
				}
				command.Fire = false
				switch rand.Intn(8) {
				case 0:
					command.Direction.Y = -1
					command.Direction.X =  0
					command.Move = true
				case 1:
					command.Direction.Y =  1
					command.Direction.X =  0
					command.Move = true
				case 2:
					command.Direction.X = -1
					command.Direction.Y =  0
					command.Move = true
				case 3:
					command.Direction.X =  1
					command.Direction.Y =  0
					command.Move = true
				}

				if rand.Intn(3) == 1 {
					if !DEBUG_DISARM_AI {
						command.Fire = true
					}
				}
			}
			commandChanel <- command
			timeEvents = time.After(time.Duration(rand.Intn(3000)) * time.Millisecond + 500)
		}
	}(commandChanel)

	return &Control{
		enabled:       false,
		commandChanel: commandChanel,
		eventChanel:   nil,
		IsPlayer: 	   false,
	}, nil
}

func (c Command) String()string  {
	return fmt.Sprintf("direction %v, moving: %v, firing: %v", c.Direction, c.Move, c.Fire)
}

type KeyBind struct {
	Up, Down, Left, Right, Fire keyboard.Key
}
