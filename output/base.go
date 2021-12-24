package output

import (
	"fmt"
	"log"
	"os"
)

var (
	buf, _ = os.OpenFile("output.log", os.O_CREATE|os.O_TRUNC, 644)
	logger = log.New(buf, "logger: ", log.Lshortfile)
)

func maxInt(x, y int) int {
	if x > y {
		return x
	}
	return y
}

func minInt(x, y int) int {
	if x < y {
		return x
	}
	return y
}

type ConsoleOutput interface {
	PrintSprite(stringer fmt.Stringer, x,y,w,h, color int) (n int, err error)
    PrintDynamicSprite(stringer fmt.Stringer, x,y,w,h, xOld,yOld,wOld,hOld, color int) (n int, err error)
	Print(str string) (n int, err error)
	MoveTo(str string, x int, y int) (out string)
	MoveCursor(x int, y int)
	Color(str string, color int) string
	Clear()
	Flush()
	Width() int
	Height() int
}

