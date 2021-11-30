package output

import (
	"fmt"
	output "github.com/buger/goterm"
	"strings"
)

func init()  {

}

type ConsoleOutput struct {
	currX, CurrY   int
	rowsRepaint    map[int]bool
	rowsRepaintCnt int
	IsFullRepaint  bool
}

func (co *ConsoleOutput) PrintSprite(stringer fmt.Stringer, x,y,color int) (n int, err error) {
	str := stringer.String()
	strH := len(strings.Split(str, "\n"))
	for i := 0; i < strH; i++ {
		co.rowsRepaint[i + y] = true
		co.rowsRepaintCnt++
	}
	str = co.MoveTo(str, x, y)
	if color > 0 {
		str = co.Color(str, color)
	}
	return output.Print(str)
}

func (co *ConsoleOutput) Print(str string) (n int, err error) {
	strH := len(strings.Split(str, "\n"))
	for i := co.CurrY; i < co.CurrY + strH; i++ {
		co.rowsRepaint[i] = true
		co.rowsRepaintCnt++
	}
	return output.Print(str)
}

func (co *ConsoleOutput) MoveTo(str string, x int, y int) (out string) {
	co.CurrY = y
	return output.MoveTo(str, x, y)
}

func (co *ConsoleOutput) MoveCursor(x int, y int) {
	co.CurrY = y
	output.MoveCursor(x, y)
}

func (co *ConsoleOutput) Color(str string, color int) string {
	return output.Color(str, color)
}

func (co *ConsoleOutput) Clear(){
	//todo try to optimize
	if co.rowsRepaintCnt < 1 {
		return
	}
	//if len(co.rowsRepaint) / co.rowsRepaintCnt > 2 {
		for index, repaint := range co.rowsRepaint {
			if repaint {
				output.MoveCursor(0, index)
				output.Print("\033[2K")
			}
			co.rowsRepaint[index] = false
		}
		co.IsFullRepaint = false
/*	} else {
		for index, _ := range co.rowsRepaint {
			co.rowsRepaint[index] = false
		}
		output.Clear()
		co.IsFullRepaint = true
	}*/
	co.rowsRepaintCnt = 0
}

func (co *ConsoleOutput) Width() int  {
	val := output.Width() //for debug
	if val <= 0 {
		val = 100
	}
	return val
}

func (co *ConsoleOutput) Height() int  {
	val := output.Height() //for debug
	if val <= 0 {
		val = 100
	}
	return val
}

func (co *ConsoleOutput) Flush(){
	output.Flush()
}

func NewConsoleOutput() (*ConsoleOutput,error)  {
	return &ConsoleOutput{
		rowsRepaint: make(map[int]bool, output.Height()),
		currX: 0,
		CurrY: 0,
	}, nil
}
