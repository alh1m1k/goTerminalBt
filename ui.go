package main

import (
	"fmt"
	direct "github.com/buger/goterm"
	"math"
	"strconv"
	"strings"
	"time"
)

type UIData struct {
	players []*Player
}

type UiSprite struct {
	Point
	zIndex int
	*Sprite
	*UIData
	TimeLeft time.Duration
}

func (receiver *UiSprite) Execute(timeLeft time.Duration) {
	buffer := receiver.Sprite.Buf
	buffer.Reset()

	xOffset := 5

	fmt.Fprint(buffer, direct.Color("Press CTRL+C to quit", direct.YELLOW))

	if receiver.UIData != nil {
		var buf, hp, ammo string
		for i, player := range receiver.UIData.players {
			if player == nil || player.Unit == nil {
				continue
			}
			if player.Unit.HP < 50 {
				hp = direct.Color(fmt.Sprintf("%03d", player.Unit.HP), direct.RED)
			} else {
				hp = direct.Color(fmt.Sprintf("%03d", player.Unit.HP), direct.CYAN)
			}
			if player.Unit.Gun.Current.Ammo == -1 {
				ammo = direct.Color("inf", direct.YELLOW)
			} else {
				ammo = direct.Color(fmt.Sprintf("%03d", player.Unit.Gun.Current.Ammo), direct.RED)
			}
			if player.Unit.destroyed && player.Retry <= 0 {
				buf = fmt.Sprintf("P%d: %s  %s",
					i+1,
					direct.Color(player.Name, direct.CYAN),
					direct.Color("IS DEAD", direct.RED),
				)
			} else {
				buf = fmt.Sprintf("P%d: %s Retry: %s Score: %05d HP: %s Ammo: %s (%s)",
					i+1,
					direct.Color(player.Name, direct.CYAN),
					direct.Color(strconv.Itoa(int(player.Retry)), direct.GREEN),
					player.Score,
					hp,
					direct.Color(player.Unit.Gun.GetName(), direct.YELLOW),
					ammo)
			}
			fmt.Fprintf(buffer, "%s %s", strings.Repeat(" ", xOffset), direct.Bold(buf))
			xOffset += 5
		}
	}

	if DEBUG {
		frameTime := timeLeft - CYCLE
		fps := 1 * time.Second / frameTime
		minFps = math.Min(float64(fps), minFps)
		maxFps = math.Max(float64(fps), maxFps)
		fmt.Fprintf(buffer, "\n %s frame time: %s fps c|mi|mx: %d | %3.2f | %3.2f", strings.Repeat(" ", 24), (frameTime).String(), fps, minFps, maxFps)
	}
}

func (receiver *UiSprite) GetXY() Point {
	return receiver.Point
}

func (receiver *UiSprite) GetSprite() Spriteer {
	return receiver.Sprite
}

func (receiver *UiSprite) GetZIndex() int {
	return receiver.zIndex
}

func NewDefaultUI() (*UiSprite, error) {
	inst := new(UiSprite)
	inst.Sprite = NewSprite()
	inst.Sprite.isAbsolute = true
	inst.Size.W, inst.Size.H = 0, 3
	//hack set w to 0 no remove clipping
	inst.zIndex = math.MaxInt32
	return inst, nil
}
