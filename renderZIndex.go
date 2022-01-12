package main

import (
	output "GoConsoleBT/output"
	"fmt"
	direct "github.com/buger/goterm"
	"math"
	"sort"
	"time"
)

type RenderZIndex struct {
	defaultZIndex    int
	defaultQueueSize int
	zIndex           []int
	needReorder      bool
	zQueue           map[int][]Renderable
	output           output.ConsoleOutput
	*UIData
	uiThrottle       *throttle
	UIDraw           bool
	offsetX, offsetY int
	total, empty     int64
}

func (receiver *RenderZIndex) Add(object Renderable) {
	var zIndex int
	if zObject, ok := object.(ZIndexed); ok {
		zIndex = zObject.GetZIndex()
	} else {
		zIndex = receiver.defaultZIndex
	}
	if receiver.zQueue[zIndex] == nil {
		receiver.zQueue[zIndex] = make([]Renderable, 0, receiver.defaultQueueSize)
		receiver.zIndex = append(receiver.zIndex, zIndex)
		receiver.needReorder = true
	}
	receiver.zQueue[zIndex] = append(receiver.zQueue[zIndex], object)
	receiver.total++
}

func (receiver *RenderZIndex) Remove(object Renderable) {
	var zIndex int
	if zObject, ok := object.(ZIndexed); ok {
		zIndex = zObject.GetZIndex()
	} else {
		zIndex = receiver.defaultZIndex
	}
	for indx, candidate := range receiver.zQueue[zIndex] {
		if object == candidate {
			receiver.zQueue[zIndex][indx] = nil
			receiver.empty++
		}
	}
}

func (receiver *RenderZIndex) Compact() {
	i, j := 0, 0
	receiver.total = 0
	receiver.empty = 0
	for _, zIndex := range receiver.zIndex {
		i, j = 0, 0
		for i < len(receiver.zQueue[zIndex]) {
			if receiver.zQueue[zIndex][i] == nil {
				//
			} else {
				receiver.zQueue[zIndex][j] = receiver.zQueue[zIndex][i]
				j++
			}
			i++
		}
		receiver.zQueue[zIndex] = receiver.zQueue[zIndex][0:j]
		receiver.total += int64(len(receiver.zQueue[zIndex]))
	}
	receiver.empty = 0
}

func (receiver *RenderZIndex) NeedCompact() bool {
	return receiver.total > 100 && receiver.empty > 0 && receiver.total/receiver.empty < 2
}

func (receiver *RenderZIndex) Execute(timeLeft time.Duration) {
	if receiver.needReorder {
		sort.Ints(receiver.zIndex)
		receiver.needReorder = false
	}
	receiver.output.Clear()
	for _, zIndex := range receiver.zIndex {
		for _, object := range receiver.zQueue[zIndex] {
			if object == nil {
				continue
			}
			sprite := object.GetSprite()
			x, y := receiver.translateXY(object.GetXY())
			wh := sprite.GetWH()
			receiver.draw(sprite, x, y, wh.W, wh.H)
		}
	}
	if receiver.UIDraw && !DEBUG_OPPORTUNITY_FIRE {
		receiver.drawUI(timeLeft)
	}
	receiver.output.MoveCursor(0, 0)
	receiver.output.Flush()
}

func (receiver *RenderZIndex) UI(data *UIData) {
	if data != nil {
		receiver.UIData = data
		receiver.UIDraw = true
	} else {
		receiver.UIDraw = false
	}
}

func (receiver *RenderZIndex) SetOffset(x, y int) {
	receiver.offsetX = x
	receiver.offsetY = y
}

func (receiver *RenderZIndex) translateXY(x, y float64) (int, int) {
	return int(math.Round(x)) + receiver.offsetX, int(math.Round(y)) + 3 + receiver.offsetY
}

func (receiver *RenderZIndex) draw(sprite Spriteer, x, y, w, h int) {
	if compose, ok := sprite.(*Composition); ok { //bypass composition position and size bugs (absolute render)
		for _, frame := range compose.frames {
			frameWh := frame.GetWH()
			receiver.output.PrintSprite(frame.Spriteer, x+frame.offsetX, y+frame.offsetY, frameWh.W, frameWh.H)
		}
	} else {
		receiver.output.PrintSprite(sprite, x, y, w, h)
	}
}

func (receiver *RenderZIndex) drawUI(timeLeft time.Duration) {

	xOffset := 0
	receiver.output.MoveCursor(0, 0)
	receiver.output.Print(receiver.output.Color("Press CTRL+C to quit", direct.YELLOW))
	xOffset += 15

	if DEBUG {
		frameTime := timeLeft - CYCLE
		fps := 1 * time.Second / frameTime
		minFps = math.Min(float64(fps), minFps)
		maxFps = math.Max(float64(fps), maxFps)
		receiver.output.Print(receiver.output.MoveTo("frame time: "+(frameTime).String(), 25, 0))
		receiver.output.Print(receiver.output.MoveTo(fmt.Sprintf("fps c|mi|mx: %d | %3.2f | %3.2f", fps, minFps, maxFps), 25, 0))
		receiver.output.Print(receiver.output.MoveTo("", 0, 1))
		receiver.output.Print(receiver.output.MoveTo("", 0, 2))
		xOffset = 55
	}

	if receiver.UIData != nil {
		var buf string
		for i, player := range receiver.UIData.players {
			if player == nil || player.Unit == nil {
				continue
			}
			buf = fmt.Sprintf("P%d: %s Retry: %d  Score: %05d HP: %03d Ammo:%s:%d",
				i+1, player.Name, player.Retry, player.Score, player.Unit.HP, player.Unit.Gun.GetName(), player.Unit.Gun.Current.Ammo)
			buf = direct.Highlight(buf, player.Name, direct.CYAN)
			receiver.output.Print(direct.Bold(receiver.output.MoveTo(buf, xOffset+10, 0)))
			xOffset += len(buf)
		}
	}
}

func NewRenderZIndex(queueSize int) (*RenderZIndex, error) {
	backend, _ := output.NewConsoleOutputLine()
	backend.CursorVisibility(false)
	backend.ClipMode(output.CLIP_MODE_RB)
	return &RenderZIndex{
		zIndex:           make([]int, 0, 5),
		zQueue:           make(map[int][]Renderable),
		needReorder:      false,
		defaultQueueSize: queueSize,
		output:           backend,
		UIData:           nil,
		uiThrottle:       newThrottle(500*time.Millisecond, true),
		UIDraw:           false,
		defaultZIndex:    100,
	}, nil
}
