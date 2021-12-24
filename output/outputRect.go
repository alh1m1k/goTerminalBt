package output

import (
	"bytes"
	"fmt"
	output "github.com/buger/goterm"
	"strings"
)

func init()  {

}

type box struct {
	x,y,w,h int
}

//convert 00 coord system to 11 coord system

type ConsoleOutputRect struct {
	currX, CurrY          int
	boxesRepaint          []box
	boxesRepaintCnt       int
	IsFullRepaint         bool
	FlushCall, SpriteCall int
}

//todo spriter category
func (co *ConsoleOutputRect) PrintSprite(stringer fmt.Stringer, x, y, w, h, color int) (n int, err error) {
	str := stringer.String()
	co.ClearRect(x,y,w,h)
	str = co.MoveTo(str, x, y)
	if color > 0 {
		str = co.Color(str, color)
	}
	co.SpriteCall++
	return output.Print(str)
}

func (co *ConsoleOutputRect) PrintSpriteDynamic(stringer fmt.Stringer, x, y, w, h, x2, y2, w2, h2, color int) (n int, err error) {
	str := stringer.String()
	co.ClearRect(x2,y2,w2,h2)
	str = co.MoveTo(str, x, y)
	if color > 0 {
		str = co.Color(str, color)
	}
	co.SpriteCall++
	return output.Print(str)
}

func (co *ConsoleOutputRect) ClearRect(x, y, w, h int) (n int, err error) {
	buffer := new(bytes.Buffer)
	for i := 0; i < h; i++ {
		buffer.Write(bytes.Repeat([]byte{' '}, w))
		buffer.Write([]byte{'\n'})
	}
	clear := co.MoveTo(buffer.String(), x, y)
	return output.Print(clear)
}

func (co *ConsoleOutputRect) clearRect(bx box) (n int, err error) {
	buffer := new(bytes.Buffer)
	for i := 0; i < bx.h; i++ {
		buffer.Write(bytes.Repeat([]byte{' '}, bx.w))
		buffer.Write([]byte{'\n'})
	}
	clear := co.MoveTo(buffer.String(), bx.x, bx.y)
	return output.Print(clear)
}

func (co *ConsoleOutputRect) Print(str string) (n int, err error) {
	strings := strings.Split(str, "\n")
	w, h := 0, len(strings)
	for i := 0; i < h; i++ {
		w = maxInt(w, len(strings[i]))
	}
	co.ClearRect(co.currX, co.CurrY, w, h)
	return output.Print(str)
}

func (co *ConsoleOutputRect) MoveTo(str string, x int, y int) (out string) {
	co.CurrY = y
	return output.MoveTo(str, x + 1, y + 1)
}

func (co *ConsoleOutputRect) MoveCursor(x int, y int) {
	co.CurrY = y
	output.MoveCursor(x + 1, y + 1)
}

func (co *ConsoleOutputRect) Color(str string, color int) string {
	return output.Color(str, color)
}

func (co *ConsoleOutputRect) CursorVisibility(visibility bool) {
	if visibility {
		output.Print("\033[?25h")
	} else {
		output.Print("\033[?25l")
	}
}

func (co *ConsoleOutputRect) Clear(){
	for _, repaint := range co.boxesRepaint {
		co.clearRect(repaint)
	}
	co.boxesRepaint = co.boxesRepaint[0:0]
	co.IsFullRepaint = false
	co.boxesRepaintCnt = 0
	co.Flush()
}

func (co *ConsoleOutputRect) Width() int  {
	val := output.Width() //for debug
	if val <= 0 {
		val = 100
	}
	return val
}

func (co *ConsoleOutputRect) Height() int  {
	val := output.Height() //for debug
	if val <= 0 {
		val = 100
	}
	return val
}

func (co *ConsoleOutputRect) Flush(){
	co.FlushCall++
	output.Output.Write(output.Screen.Bytes())
	output.Output.Flush()
	output.Screen.Reset()
}

func NewConsoleOutputRect() (*ConsoleOutputRect,error)  {
	return &ConsoleOutputRect{
		boxesRepaint: make([]box, output.Height()),
		currX: 0,
		CurrY: 0,
	}, nil
}
