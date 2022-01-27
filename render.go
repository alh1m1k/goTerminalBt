package main

import (
	output "GoConsoleBT/output"
	direct "github.com/buger/goterm"
	"math"
	"sort"
	"strconv"
	"time"
)

type Renderable interface {
	GetXY() Point
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
	SetOffset(x, y int)
	NeedCompact() bool
	Compact()
	Free()
}

var minFps float64 = math.MaxFloat64
var maxFps float64 = 0

type Render struct {
	defaultZIndex    int
	defaultQueueSize int
	zIndex           []int
	needReorder      bool
	zQueue           map[int][]Renderable
	output           output.ConsoleOutput
	UIDraw           bool
	offsetX, offsetY int
	total, empty     int64
}

func (receiver *Render) Add(object Renderable) {
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

func (receiver *Render) Remove(object Renderable) {
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

func (receiver *Render) Compact() {
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

func (receiver *Render) NeedCompact() bool {
	return receiver.total > 100 && receiver.empty > 0 && receiver.total/receiver.empty < 2
}

func (receiver *Render) Execute(timeLeft time.Duration) {
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
			info := sprite.GetInfo()
			x, y := receiver.translateXY(object.GetXY(), info.isAbsolute)

			receiver.draw(sprite, x, y, info.Size.W, info.Size.H)
			if DEBUG_SHOW_ID {
				if oi, ok := object.(ObjectInterface); ok {
					receiver.output.Print(receiver.output.MoveTo(" "+receiver.output.Color(strconv.Itoa(int(oi.GetAttr().ID)), direct.CYAN)+" ", x, y))
				}
			}
			if DEBUG_SHOW_AI_BEHAVIOR {
				if oi, ok := object.(*Unit); ok && oi.GetAttr().AI {
					receiver.output.Print(receiver.output.MoveTo(" "+receiver.output.Color(oi.Control.(*BehaviorControl).Behavior.Name(), direct.CYAN)+" ", x, y+1))
				}
			}
		}
	}
	receiver.output.MoveCursor(0, 0)
	receiver.output.Flush()
}

func (receiver *Render) SetOffset(x, y int) {
	receiver.offsetX = x
	receiver.offsetY = y
}

func (receiver *Render) Free() {
	receiver.output.CursorVisibility(true)
	receiver.zQueue = make(map[int][]Renderable)
}

func (receiver *Render) translateXY(pos Point, absolute bool) (int, int) {
	if absolute {
		return int(math.Round(pos.X)), int(math.Round(pos.Y))
	} else {
		return int(math.Round(pos.X)) + receiver.offsetX, int(math.Round(pos.Y)) + receiver.offsetY
	}
}

func (receiver *Render) draw(sprite Spriteer, x, y, w, h int) {
	if compose, ok := sprite.(*Composition); ok { //bypass composition position and size bugs (absolute render)
		for _, frame := range compose.frames {
			frameWh := frame.GetInfo().Size
			receiver.output.PrintSprite(frame.Spriteer, x+frame.offsetX, y+frame.offsetY, frameWh.W, frameWh.H)
		}
	} else {
		receiver.output.PrintSprite(sprite, x, y, w, h)
	}
}

func NewRenderZIndex(queueSize int) (*Render, error) {
	backend, _ := output.NewConsoleOutputLine()
	backend.CursorVisibility(false)
	backend.ClipMode(output.CLIP_MODE_RB)
	return &Render{
		zIndex:           make([]int, 0, 5),
		zQueue:           make(map[int][]Renderable),
		needReorder:      false,
		defaultQueueSize: queueSize,
		output:           backend,
		UIDraw:           false,
		defaultZIndex:    100,
	}, nil
}
