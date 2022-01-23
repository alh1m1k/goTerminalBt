package output

import (
	"fmt"
	"log"
	"os"
)

const (
	CLIP_MODE_NONE = iota
	CLIP_MODE_LT  		//left top
	CLIP_MODE_RB		//right bottom
)

var (
	buf, _ = os.OpenFile("./output.log", os.O_CREATE|os.O_TRUNC, 644)
	logger = log.New(buf, "logger: ", log.Lshortfile)
	DEBUG = false
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
	PrintSprite(stringer fmt.Stringer, x, y, w, h int) (n int, err error)
    PrintDynamicSprite(stringer fmt.Stringer, x, y, w, h, xOld, yOld, wOld, hOld int) (n int, err error)
	Print(str string) (n int, err error)
	MoveTo(str string, x int, y int) (out string)
	MoveCursor(x int, y int)
	CursorVisibility(visibility bool)
	ClipMode(mode int)
	Color(str string, color int) string
	Clear()
	Flush()
	Width() int
	Height() int
}

