package output

import (
	"fmt"
	output "github.com/buger/goterm"
	"strings"
)


//convert 00 coord system to 11 coord system

//failsave render backend for windows terminal

type ConsoleOutputBitmap struct {
	currX, CurrY          int
	bitmap          	  [][]int
	repaintIndex		  int
	boxesRepaintCnt       int
	IsFullRepaint         bool
	FlushCall, SpriteCall int
}

//todo spriter category
func (co *ConsoleOutputBitmap) PrintSprite(stringer fmt.Stringer, x, y, w, h, color int) (n int, err error) {
	str := stringer.String()
	co.ClearRect(x, y, w, h)
	str = co.MoveTo(str, x, y)
	if color > 0 {
		str = co.Color(str, color)
	}
	co.SpriteCall++
	return output.Print(str)
}

func (co *ConsoleOutputBitmap) PrintSpriteDynamic(stringer fmt.Stringer, x, y, w, h, x2, y2, w2, h2, color int) (n int, err error) {
	str := stringer.String()
	co.ClearRect(x, y, w, h)
	str = co.MoveTo(str, x, y)
	if color > 0 {
		str = co.Color(str, color)
	}
	co.SpriteCall++
	return output.Print(str)
}

func (co *ConsoleOutputBitmap) Print(str string) (n int, err error) {
	strings := strings.Split(str, "\n")
	w, h := 0, len(strings)
	for i := 0; i < h; i++ {
		w = maxInt(w, len(strings[i]))
	}
	co.ClearRect(co.currX, co.CurrY, w, h)
	return output.Print(str)
}

func (co *ConsoleOutputBitmap) ClearRect(x, y, w, h int) {
	maxW, maxH := minInt(len(co.bitmap[0]) - 1, x + w) , minInt(len(co.bitmap) - 1, y + h)
	for i := y; i < maxH; i++ {
		for j := x; j < maxW; j++ {
			if co.bitmap[i][j] != co.repaintIndex {
				co.MoveCursor(j, i)
				output.Print(" ")
				co.bitmap[i][j] = co.repaintIndex
			}
		}
	}
}

func (co *ConsoleOutputBitmap) MoveTo(str string, x int, y int) (out string) {
	co.CurrY = y
	return output.MoveTo(str, x + 1, y + 1)
}

func (co *ConsoleOutputBitmap) MoveCursor(x int, y int) {
	co.CurrY = y
	output.MoveCursor(x + 1, y + 1)
}

func (co *ConsoleOutputBitmap) Color(str string, color int) string {
	return output.Color(str, color)
}

func (co *ConsoleOutputBitmap) CursorVisibility(visibility bool) {
	if visibility {
		output.Print("\033[?25h")
	} else {
		output.Print("\033[?25l")
	}
}

func (co *ConsoleOutputBitmap) Clear(){
	co.repaintIndex++
}

func (co *ConsoleOutputBitmap) Width() int  {
	val := output.Width() //for debug
	if val <= 0 {
		val = 100
	}
	return val
}

func (co *ConsoleOutputBitmap) Height() int  {
	val := output.Height() //for debug
	if val <= 0 {
		val = 100
	}
	return val
}

func (co *ConsoleOutputBitmap) Flush(){
/*	start 	:= -1
	length 	:= 0*/
	/*for i := 0; i < len(co.bitmap); i++ {
		if length > 0 {
			co.MoveCursor(start, i - 1)
			output.Print(strings.Repeat(" ", len(co.bitmap[i-1]) - start))
			length = 0
			start  = -1
		}
		for j := 0; j < len(co.bitmap[i]); j++ {
			if co.bitmap[i][j] - co.repaintIndex == -1 { //on bit that outdate at one frame
				if start == -1 {
					start = j
				}
				length++
 			} else if length > 0 {
				co.MoveCursor(start, i)
				output.Print(strings.Repeat(" ", length))
				length = 0
				start  = -1
			}
		}
	}*/
	for i := 0; i < len(co.bitmap); i++ {
		for j := 0; j < len(co.bitmap[i]); j++ {
			if co.bitmap[i][j] - co.repaintIndex == -1 { //on bit that outdate at one frame
				co.MoveCursor(j, i)
				output.Print(" ")
			}
		}
	}
	co.FlushCall++
	output.Output.Write(output.Screen.Bytes())
	output.Output.Flush()
	output.Screen.Reset()
}

func NewConsoleOutputBitmap() (*ConsoleOutputBitmap,error)  {
	instance := &ConsoleOutputBitmap{
		bitmap: make([][]int, output.Height()),
		currX: 0,
		CurrY: 0,
	}
	for index := range instance.bitmap {
		instance.bitmap[index] = make([]int, output.Width())
	}
	return instance, nil
}
