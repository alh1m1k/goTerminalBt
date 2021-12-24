package output

import (
	"fmt"
	output "github.com/buger/goterm"
	"strings"
)

func init()  {

}

//convert 00 coord system to 11 coord system

type ConsoleOutputLine2 struct {
	currX, CurrY   int
	rowsRepaint    map[int]int
	repaintIndex   int
	rowsRepaintCnt int
	IsFullRepaint  bool
	FlushCall, SpriteCall int
}

func (co *ConsoleOutputLine2) PrintSprite(stringer fmt.Stringer, x,y,w,h, color int) (n int, err error) {
	str := stringer.String()
	co.ClearRect(x,y,w,h)
	str = co.MoveTo(str, x, y)
	if color > 0 {
		str = co.Color(str, color)
	}
	return output.Print(str)
}

func (co *ConsoleOutputLine2) PrintDynamicSprite(stringer fmt.Stringer, x,y,w,h, color int) (n int, err error) {
	str := stringer.String()
	co.ClearRect(x,y,w,h)
	str = co.MoveTo(str, x, y)
	if color > 0 {
		str = co.Color(str, color)
	}
	return output.Print(str)
}

func (co *ConsoleOutputLine2) ClearRect(x, y, w, h int) {
	for i := 0; i < h; i++ {
		if co.rowsRepaint[i + y] != co.repaintIndex {
			co.rowsRepaint[i + y] = co.repaintIndex
			co.rowsRepaintCnt++
			output.MoveCursor(1, i + y + 1)
			output.Print("\033[2K")
		}
	}
}

func (co *ConsoleOutputLine2) Print(str string) (n int, err error) {
	strH := len(strings.Split(str, "\n"))
	co.ClearRect(co.currX,co.CurrY,0,strH)
	return output.Print(str)
}

func (co *ConsoleOutputLine2) MoveTo(str string, x int, y int) (out string) {
	co.CurrY = y
	return output.MoveTo(str, x + 1, y + 1)
}

func (co *ConsoleOutputLine2) MoveCursor(x int, y int) {
	co.CurrY = y
	output.MoveCursor(x + 1, y + 1)
}

func (co *ConsoleOutputLine2) Color(str string, color int) string {
	return output.Color(str, color)
}

func (co *ConsoleOutputLine2) Clear(){
	co.repaintIndex++
}

func (co *ConsoleOutputLine2) Width() int  {
	val := output.Width() //for debug
	if val <= 0 {
		val = 100
	}
	return val
}

func (co *ConsoleOutputLine2) Height() int  {
	val := output.Height() //for debug
	if val <= 0 {
		val = 100
	}
	return val
}

func (co *ConsoleOutputLine2) Flush(){
	for index, repaint := range co.rowsRepaint {
		if repaint != co.repaintIndex {
			output.MoveCursor(1, index + 1)
			output.Print("\033[2K")
		}
		co.rowsRepaint[index] = 0
	}
	output.Output.Write(output.Screen.Bytes())
	output.Output.Flush()
	output.Screen.Reset()
}

func NewConsoleOutputLine2() (*ConsoleOutputLine2,error)  {
	return &ConsoleOutputLine2{
		rowsRepaint: make(map[int]int, output.Height()),
		currX: 0,
		CurrY: 0,
	}, nil
}
