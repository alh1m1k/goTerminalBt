# GoConsoleBT
![Alt-текст](/playerMenu.png "Menu")

It's Battle City inspired game for console and terminal. Game use text 
and ASCII control symbol as graphics. GoConsoleBT is a game engine and game prototype written fully on GO

### Platform
Currently tested on Windows 10 and Debian 10. Use any modern terminal to start
(cmd.exe not a case) on windows you may run on Windows Terminal

### Installation
Install and update this app with `go get -u github.com/alh1m1k/goTerminalBt/blob/main/`
then install all requirements, on linux platform also need install libasound2-dev `apt install libasound2-dev`
run from app directory `go build . ` to build app, or `go run . ` to build and run (go lang must be installed)

### Usage

![Alt-текст](/stage1Normal.png "Stage-1")

First of all, build game for platform you want. 
then run terminal to start game, use --help to see all available opt.
+ --scenario run custom scenario (for now only one game scenario available: stage-1 `app --scenario stage-1` )  
or use:  
--tankCnt --wallCnt to run buildin random scenario with specified enemy and obstacle count `app --wallCnt 250 --tankCnt 30` (by default total wallCnt+tankCnt must be less then 300-350)
+ --withColor enable opt. color mode
+ --withSound enable opt. sound support (sound will play on machine where game actually run)
+ --simplifyAl disabling behavioral ai and switching to random (behavior ai is kinda buggy for now)

After startup, game save config and restart, when you see screen configurator
![Alt-текст](/configurate.png "Cfg") zoom out until you see border.
Game border control by config, you may increase it, but I don't recommend reducing it. Then Press Enter and game will start.

### Controls
By Default player-1 use `arrow keys` and `space` to fire, player-2 use `wsad` and `backspace` to fire

###  Uses and thanks
Thank's to author's there libs

| Author | Hub | Uses |
|----------------|:---------:|----------------:|
| tanema | https://github.com/tanema/ump | collision engine |
| buger | https://github.com/buger/goterm | terminal backend |
| buger | https://github.com/buger/jsonparser | json parser |
| eiannone | https://github.com/eiannone/keyboard | keyboard input |
| faiface | https://github.com/faiface/beep | sound engine |
| xarg | https://github.com/xarg/gopathfinding | A* impl |
| xiaonanln | https://github.com/xiaonanln/go-lockfree-pool | pool |

me for the bugs...

## Problems
+ Glitch ai behavior 
+ Rare crash on closing

### Disclaimer 
I'm not a game developer and not a Go developer 


#### Random with color
![Alt-текст](/withColor.png "Colorfull")
