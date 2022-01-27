package output

import (
	"errors"
	"fmt"
	output "github.com/buger/goterm"
	"io"
	"log"
	"os"
	"strings"
	"time"
)

func init()  {

}

var (
	Output  = os.Stdout
	OutOfRenderRangeError = errors.New("out of render range")
	blinkFixer  = log.New(os.Stderr, "", 0)
)

//convert 00 coord system to 11 coord system

type ConsoleOutputLine struct {
	currX, currY    int
	rowsRepaint     []bool
	rowsRepaintCnt  int
	needFullRepaint bool
	clipMode  int
	width, height, wTolerance, hTolerance int
}

func (co *ConsoleOutputLine) PrintSprite(stringer fmt.Stringer, x, y, w, h int) (n int, err error) {
	if co.clipTest(x,y,w,h) {
		if DEBUG {
			log.Print("\n clip: ", x,y,w,h, output.Width(), output.Height())
		}
		return 0, OutOfRenderRangeError
	}
	str := stringer.String()
	for i := maxInt(y, 0); i < y + h; i++ {
		co.rowsRepaint[i] = true
		co.rowsRepaintCnt++
	}
	str = co.MoveTo(str, x, y)
	return output.Print(str)
}

func (co *ConsoleOutputLine) PrintDynamicSprite(stringer fmt.Stringer, x, y, w, h, xOld, yOld, wOld, hOld int) (n int, err error) {
	if co.clipTest(x,y,w,h) {
		if DEBUG {
			log.Print("\n clip: ", x,y,w,h, output.Width(), output.Height())
		}
		return 0, OutOfRenderRangeError
	}
	str := stringer.String()
	for i := maxInt(y, 0); i < y + h; i++ {
		co.rowsRepaint[i] = true
		co.rowsRepaintCnt++
	}
	str = co.MoveTo(str, x, y)
	return output.Print(str)
}

func (co *ConsoleOutputLine) Print(str string) (n int, err error) {
	strH := len(strings.Split(str, "\n"))
	if co.clipTest(0, co.currY,0, strH) {
		if DEBUG {
			log.Print("\n clip: ", co.currY + strH, output.Height())
		}
		return 0, OutOfRenderRangeError
	}
	for i := maxInt(co.currY, 0); i < co.currY+ strH; i++ {
		co.rowsRepaint[i] = true
		co.rowsRepaintCnt++
	}
	return output.Print(str)
}

func (co *ConsoleOutputLine) clipTest(x, y, w, h int) bool {
	if co.clipMode == CLIP_MODE_NONE {
		return false
	}
	var b1, b2 int
	if co.clipMode == CLIP_MODE_LT {
		b1, b2 = x, y
	} else {
		b1, b2 = x + w, y + h
	}
	if b1 < 0 || b2 < 0 || b1 > co.width + co.wTolerance || b2 > co.height + co.hTolerance {
		return true
	}
	return false
}

func (co *ConsoleOutputLine) ClipMode(mode int) {
	co.clipMode = mode
}

func (co *ConsoleOutputLine) MoveTo(str string, x int, y int) (out string) {
	co.currY = y
	return output.MoveTo(str, x + 1, y + 1)
}

func (co *ConsoleOutputLine) MoveCursor(x int, y int) {
	co.currY = y
	output.MoveCursor(x + 1, y + 1)
}

func (co *ConsoleOutputLine) CursorVisibility(visibility bool) {
	if visibility {
		output.Print("\033[?25h")
	} else {
		output.Print("\033[?25l")
	}
}

func (co *ConsoleOutputLine) Color(str string, color int) string {
	return output.Color(str, color)
}

func (co *ConsoleOutputLine) Clear(){
	if co.needFullRepaint {
		output.Clear()
		co.needFullRepaint = false
		return
	}
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
	//print some to stderr // remove blink totaly
	blinkFixer.Print(" ")
	//bypass original flush op due blink issue //significantly reduce blink effect
	io.Copy(output.Output, output.Screen)
	output.Screen.Reset()
	go output.Output.Flush()
}

func NewConsoleOutputLine() (*ConsoleOutputLine,error)  {
	instance := &ConsoleOutputLine{
		currX:           0,
		currY:           0,
		rowsRepaint:     make([]bool, output.Height() + 3),
		rowsRepaintCnt:  0,
		needFullRepaint: false,
		width:           0,
		height:          0,
		wTolerance:      3,
		hTolerance:      3,
	}
	updateSizesDispatcher(instance)
	return instance, nil
}

func updateSizesDispatcher(cOut *ConsoleOutputLine)  {
	var check func()
	check = func() {
		w, h := output.Width(), output.Height()
		withTolerance := h + cOut.hTolerance
		if cOut.width != w || cOut.height != h {
			if rLen := len(cOut.rowsRepaint); rLen < withTolerance {
				cOut.rowsRepaint = append(cOut.rowsRepaint, make([]bool, withTolerance - rLen)...)
			} else if rLen > h {
				cOut.rowsRepaint = cOut.rowsRepaint[:withTolerance]
			}
			cOut.width, cOut.height = w,h
			cOut.needFullRepaint = true
		}
		time.AfterFunc(time.Second / 2, check)
	}
	check()
}
