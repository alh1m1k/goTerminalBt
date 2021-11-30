package main

import (
	output "GoConsoleBT/output"
	"fmt"
	direct "github.com/buger/goterm"
	"math"
	"time"
)

type Renderable interface {
	GetXY() (x float64, y float64)
	GetSprite() Spriteer
}

type ZIndexed interface {
	GetZIndex() int
	//todo updateZIndexCb
}

type Renderer interface {
	Add(object Renderable)
	Remove(object Renderable)
	Execute(timeLeft time.Duration)
	UI(data *UIData)
	SetOffset(x, y int)
}

var minFps float64 = math.MaxFloat64
var maxFps float64 = 0

type UIData struct {
	players  []*Player
}

type Render struct {
	queue 	[]Renderable
	output 	*output.ConsoleOutput
	*UIData
	uiThrottle       *throttle
	UIDraw           bool
	offsetX, offsetY int
}

func (receiver *Render) Add(object Renderable) {
	receiver.queue = append(receiver.queue, object)
}

func (receiver *Render) Remove(object Renderable) {
	for indx, candidate := range receiver.queue {
		if object == candidate {
			receiver.queue[indx] = nil
		}
	}
}

func (receiver *Render) Execute(timeLeft time.Duration) {
	receiver.output.Clear()
	for _, object := range receiver.queue {
		if object == nil {
			continue
		}
		sprite := object.GetSprite()
		x, y := receiver.translateXY(object.GetXY())
		receiver.draw(sprite, x, y)
	}

	if (receiver.uiThrottle.Reach(timeLeft) || receiver.output.IsFullRepaint) && receiver.UIDraw {
		receiver.drawUI(timeLeft)
	}
	receiver.output.MoveCursor(0, 0)
	receiver.output.Flush()
}

func (receiver *Render) UI(data *UIData)  {
	if data != nil {
		receiver.UIData = data
		receiver.UIDraw = true
	} else {
		receiver.UIDraw = false
	}
}

func (receiver *Render) SetOffset(x,y int)  {
	receiver.offsetX = x
	receiver.offsetY = y
}

func (receiver *Render) translateXY(x, y float64) (int, int) {
	return int(x) + 0, int(y) + 3
}

func (receiver *Render) draw(sprite Spriteer, x, y int) {

	receiver.output.PrintSprite(sprite, x, y, 0)
}

func (receiver *Render) drawUI(timeLeft time.Duration) {
	frameTime := timeLeft - CYCLE
	fps := 1 * time.Second / frameTime
	minFps = math.Min(float64(fps), minFps)
	maxFps = math.Max(float64(fps), maxFps)
	direct.MoveCursor(0,0)
	direct.Println(direct.Color("Press CTRL+C to quit", direct.YELLOW))
	direct.Print(direct.MoveTo("frame time: " + (frameTime).String(), 25, 0))
	direct.Print(direct.MoveTo(fmt.Sprintf("fps c|mi|mx: %d | %3.2f | %3.2f", fps, minFps, maxFps), 25, 0))
	if receiver.UIData != nil {
		var buf string
		var xOffset = 55
		for i, player := range receiver.UIData.players {
			if player == nil || player.Unit == nil {
				continue
			}
			buf = fmt.Sprintf("P%d: %s Retry: %d  Score: %05d HP: %03d Ammo:%s:%d",
				i+1, player.Name, player.Retry, player.Score, player.Unit.HP, player.Unit.Gun.GetName(), player.Unit.Gun.Current.Ammo)
			buf = direct.Highlight(buf, player.Name, direct.CYAN)
			direct.Print(direct.Bold(direct.MoveTo(buf, xOffset + 10, 0)))
			xOffset += len(buf)
		}
	}
}

func NewRender(queueSize int) (*Render, error) {
	output, _ := output.NewConsoleOutput()
	return &Render{
		queue:     make([]Renderable, 0, queueSize),
		output:     output,
		UIData:     nil,
		uiThrottle: newThrottle(500*time.Millisecond, true),
		UIDraw:     false,
	}, nil
}
