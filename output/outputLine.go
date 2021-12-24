package output

import (
	"bufio"
	"fmt"
	output "github.com/buger/goterm"
	"strings"
)

func init()  {

}

var Output *bufio.Writer = nil

//convert 00 coord system to 11 coord system

type ConsoleOutputLine struct {
	currX, CurrY   int
	rowsRepaint    map[int]bool
	rowsRepaintCnt int
}

func (co *ConsoleOutputLine) PrintSprite(stringer fmt.Stringer, x,y,w,h, color int) (n int, err error) {
	str := stringer.String()
	for i := 0; i < h; i++ {
		co.rowsRepaint[i + y] = true
		co.rowsRepaintCnt++
	}
	str = co.MoveTo(str, x, y)
	if color > 0 {
		str = co.Color(str, color)
	}
	return output.Print(str)
}

func (co *ConsoleOutputLine) PrintDynamicSprite(stringer fmt.Stringer, x,y,w,h, x2,y2,w2,h2, color int) (n int, err error) {
	str := stringer.String()
	for i := 0; i < h; i++ {
		co.rowsRepaint[i + y] = true
		co.rowsRepaintCnt++
	}
	str = co.MoveTo(str, x, y)
	if color > 0 {
		str = co.Color(str, color)
	}
	return output.Print(str)
}

func (co *ConsoleOutputLine) Print(str string) (n int, err error) {
	strH := len(strings.Split(str, "\n"))
	for i := co.CurrY; i < co.CurrY + strH; i++ {
		co.rowsRepaint[i] = true
		co.rowsRepaintCnt++
	}
	return output.Print(str)
}

func (co *ConsoleOutputLine) MoveTo(str string, x int, y int) (out string) {
	co.CurrY = y
	return output.MoveTo(str, x + 1, y + 1)
}

func (co *ConsoleOutputLine) MoveCursor(x int, y int) {
	co.CurrY = y
	output.MoveCursor(x + 1, y + 1)
}

func (co *ConsoleOutputLine) Color(str string, color int) string {
	return output.Color(str, color)
}

func (co *ConsoleOutputLine) Clear(){
	if co.rowsRepaintCnt < 1 {
		return
	}
	for index, repaint := range co.rowsRepaint {
		if repaint {
			output.MoveCursor(1, index+1)
			output.Print("\033[2K")
		}
		co.rowsRepaint[index] = false
	}
	co.rowsRepaintCnt = 0
}

/**
func (co *ConsoleOutputLine) Clear(){
	if co.rowsRepaintCnt < 1 {
		return
	}
	if len(co.rowsRepaint) / co.rowsRepaintCnt >= 2 {
		for index, repaint := range co.rowsRepaint {
			if repaint {
				output.MoveCursor(1, index + 1)
				output.Print("\033[2K")
			}
			co.rowsRepaint[index] = false
		}
		co.IsFullRepaint = false
	} else {
		for index, _ := range co.rowsRepaint {
			co.rowsRepaint[index] = false
		}
		output.Clear()
		co.IsFullRepaint = true
	}
	co.rowsRepaintCnt = 0
}
 */

func (co *ConsoleOutputLine) Width() int  {
	val := output.Width() //for debug
	if val <= 0 {
		val = 100
	}
	return val
}

func (co *ConsoleOutputLine) Height() int  {
	val := output.Height() //for debug
	if val <= 0 {
		val = 100
	}
	return val
}

func (co *ConsoleOutputLine) Flush(){
	//bypass original flush op due blink issue
	output.Output.Write(output.Screen.Bytes())
	output.Output.Flush()
	output.Screen.Reset()
}

func NewConsoleOutputLine() (*ConsoleOutputLine,error)  {
	return &ConsoleOutputLine{
		rowsRepaint: make(map[int]bool, output.Height()),
		currX: 0,
		CurrY: 0,
	}, nil
}
