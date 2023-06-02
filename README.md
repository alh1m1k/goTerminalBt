# GoConsoleBT
![Alt-текст](/playerMenu.png "Menu")

This is a terminal game inspired by the battle city. Game use text
and ASCII control symbols as graphics. GoConsoleBT is a game engine and game prototype written fully on GO

### Platform
Currently tested on Windows 10 and Debian 10. Use any modern terminal to start
(cmd.exe not a case) on windows you may run on Windows Terminal

### Installation
Install and update this app with `go get -u github.com/alh1m1k/goTerminalBt/blob/main/`
then install all requirements, on linux platform also need install libasound2-dev `apt install libasound2-dev`.
Run from app directory `go build . ` to build app, or `go run . ` to build and run (go lang must be installed)

![Alt-текст](/stage1Normal.png "Stage-1")

### Usage
First of all, build game for platform you want. On fastest way is: 
```sh
 docker run --rm -it -v  <project-dir>:/usr/src/myapp -w /usr/src/myapp golang bash
'''
in container shell
```sh
 apt update && apt install libasound2-dev
```
then 
```sh
 go build .
```
then run terminal to start game, use --help to see all available opt.
+ --scenario run custom scenario (for now only one game scenario available: stage-1 `app --scenario stage-1` )  
or use:  
--tankCnt --wallCnt to run buildin random scenario with specified enemy and obstacle count `app --wallCnt 250 --tankCnt 30` (by default total wallCnt+tankCnt must be less then 300-350)
+ --withColor enable opt. color mode
+ --withSound enable opt. sound support (sound will play on machine where game actually run)
+ --simplifyAl disabling behavioral ai and switching to random (behavior ai is kinda buggy for now)

After startup, game will save config and restart, then you see the screen configurator
![Alt-текст](/configurate.png "Cfg") zoom out until you can see the border.
Game border controlled by config, you may increase it, but I don't recommend reducing it. Then Press Enter and game will start.

### Controls
By Default player-1 use `arrow keys` and `space` to fire, player-2 use `wsad` and `backspace` to fire

### Sound
This repository do not contain any sound's. If you need them, look `./sounds/readme.txt`

###  Uses and thanks
Thanks to the authors of libraries

| Author | Hub | Uses |
|----------------|:---------:|----------------:|
| tanema | https://github.com/tanema/ump | collision engine |
| buger | https://github.com/buger/goterm | terminal backend |
| buger | https://github.com/buger/jsonparser | json parser |
| eiannone | https://github.com/eiannone/keyboard | keyboard input |
| faiface | https://github.com/faiface/beep | sound engine |
| xarg | https://github.com/xarg/gopathfinding | A* impl |
| xiaonanln | https://github.com/xiaonanln/go-lockfree-pool | pool |

and me for the game bugs...

## Problems
+ It's hard
+ Glitch ai behavior
+ Rare crash on closing
+ Flickering when many unit displays on screen (only for Windows Terminal), try to reduce unit count or disable color

### Disclaimer 
I'm not a game developer and not a Go developer 

#### Random with color
![Alt-текст](/withColor.png "Colorfull")
