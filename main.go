package main

import (
	"GoConsoleBT/collider"
	"GoConsoleBT/controller"
	"flag"
	direct "github.com/buger/goterm"
	"github.com/eiannone/keyboard"
	"github.com/pkg/profile"
	"log"
	"math"
	"math/rand"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"syscall"
	"time"
)

const CYCLE = 100 * time.Millisecond
const SLOW_CYCLE = time.Second / 2
const TIME_FACTOR = time.Second / CYCLE

const DEBUG = false
const DEBUG_SPAWN = false
const DEBUG_EVENT = false
const DEBUG_EXEC = false
const DEBUG_STATE = false
const DEBUG_NO_AI = false
const DEBUG_SHAKE = false
const DEBUG_IMMORTAL_PLAYER = true
const DEBUG_FREEZ_AI = true
const DEBUG_AI_PATH = true
const DEBUG_AI_BEHAVIOR = false
const DEBUG_FIRE_SOLUTION = false
const DEBUG_MINIMAP = true

const RENDERER_WITH_ZINDEX = true

var (
	buf, _          = os.OpenFile("log.txt", os.O_CREATE|os.O_TRUNC, 644)
	logger          = log.New(buf, "logger: ", log.Lshortfile)
	profilerHandler interface {
		Stop()
	}
	CycleID int64 = 0
)

var (
	gameConfig           *GameConfig
	game                 *Game
	render               Renderer
	calibration          *Calibration
	endGame              = false
	endGameThrottle      = newThrottle(3*time.Second, false)
	endGameCycleThrottle = newThrottle(1*time.Second, true)
	AIBUILDER            func() (*BehaviorControl, error)
)

//flags
var (
	seed             int64
	wallCnt, tankCnt int
	calibrate        bool
	lockfreePool     bool
	profileMod       string
	scenarioName     string
	profileDelay     time.Duration
	osSignal         chan os.Signal
)

func init() {
	dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		log.Fatal(err)
	}
	profile.ProfilePath(dir)
	flag.Int64Var(&seed, "seed", time.Now().UnixNano(), "random generator seed")
	flag.StringVar(&scenarioName, "scenario", "random", "run scenario")
	flag.IntVar(&tankCnt, "tankCnt", 25, "for random scenario")
	flag.IntVar(&wallCnt, "wallCnt", 25, "for random scenario")
	flag.BoolVar(&calibrate, "calibrate", false, "terminal calibration mode")
	flag.StringVar(&profileMod, "profile.mode", "", "enable profiling mode, one of [cpu, mem, mutex, block, all]")
	flag.DurationVar(&profileDelay, "profile.delay", -1, "delay of starting profile, after game start. -1 means no delay")

	osSignal = make(chan os.Signal, 1)

	//cause problem on debug?
	signal.Notify(osSignal, syscall.SIGTERM, syscall.SIGQUIT, syscall.SIGKILL, syscall.SIGINT)
}

func main() {

	//EffectAnimDisappear("stealth/tank/left/tank", 16, 42)
	//EffectAnimInterference("stealth/tank/bottom/tank", 10, 0.3)

	//EffectVerFlip(bytes.NewReader([]byte("123\n456\n789")), os.Stdout)
	/*	EffectAnimNormalizeNewLine("stealth/tank/right/tank", 17)
		EffectAnimVerFlip("stealth/tank/right/tank", 17)*/

	/*	EffectAnimNormalizeNewLine("flak/left/flak", 10)
		EffectAnimVerFlip("flak/left/flak", 10)

		os.Exit(0)*/

	flag.Parse()

	rand.Seed(seed)

	var err error
	gameConfig, err = loadConfig()
	if err != nil {
		if !calibrate {
			panic("no config found, run --calibrate")
		} else {
			gameConfig, _ = NewDefaultGameConfig()
		}
	}

	flag.Parsed()

	//input
	keysEvents, err := keyboard.GetKeys(1)
	if err != nil {
		panic(err)
	}
	defer func() {
		_ = keyboard.Close()
		profileStop()
	}()

	//start pipeline
	pipe, _ := NewGPipeline()

	//animation
	animator, _ := getAnimationManager()
	pipe.AnimationManager = animator

	//render
	if RENDERER_WITH_ZINDEX {
		render, _ = NewRenderZIndex(100)
	} else {
		render, _ = NewRender(100)
	}
	pipe.Render = render

	//updater
	updater, _ := NewUpdater(100)
	pipe.Updater = updater

	//effects
	pipe.EffectManager, _ = NewEffectManager(render, updater)

	//collider
	detector, _ := collider.NewCollider(100)
	pipe.Collider = detector

	//vision
	vision, _ := NewVisioner(detector, 100)
	pipe.Visioner = vision

	//Location
	location, _ := NewLocation(Point{
		X: gameConfig.Box.X,
		Y: gameConfig.Box.Y,
	}, Point{
		X: gameConfig.Box.W / gameConfig.ColWidth,
		Y: gameConfig.Box.H / gameConfig.RowHeight,
	})
	detector.Add(location)
	pipe.Location = location

	//spawner
	spawner, _ := NewSpawner(updater, render, detector, location, vision, gameConfig)
	pipe.SpawnManager = spawner

	navigation, _ := NewNavigation(location, detector)
	pipe.Navigation = navigation

	//builder
	buildManager, _ := NewBlueprintManager()
	buildManager.AddLoaderPackage(NewJsonPackage())
	buildManager.GameConfig = gameConfig
	buildManager.EventChanel = spawner.UnitEventChanel //remove from builder

	//ai
	aibuilder, _ := NewAIControlBuilder(detector, location, navigation)
	AIBUILDER = aibuilder.Build
	//AIBUILDER()

	//scenario
	scenario, _ := /*NewCollisionDemoScenario(tankCnt, wallCnt)*/ NewRandomScenario(tankCnt, wallCnt)
	scenario.DeclareBlueprint(func(blueprint string) {
		builder, _ := buildManager.CreateBuilder(blueprint)
		if builder == nil { //may cause error on success
			panic("builder " + blueprint + " not found")
		} else {
			spawner.AddBuilder(blueprint, builder)
		}
		object, _ := buildManager.Get(blueprint)
		if projectile, ok := object.(*Projectile); ok {
			if err := aibuilder.RegisterProjectile(projectile); err != nil {
				logger.Println(err)
			}
		}
	})

	//game
	game, _ = NewGame(nil, spawner)
	game.Location = location
	game.EffectManager = pipe.EffectManager

	//time
	cycleTime := CYCLE
	var timeCurrent time.Time = time.Now()
	var timeLeft time.Duration
	timeEvents := time.After(cycleTime)

	direct.Clear()
	direct.MoveCursor(0, 0)
	direct.Flush()

	var terminateEvent, resultScreen <-chan time.Time
	var screen Screener
	var configurationChanel EventChanel = make(EventChanel)

	if calibrate {
		calibration, _ = NewCalibration(updater, render, detector, location, configurationChanel)
		calibration.GameConfig = gameConfig
		control, _ := controller.NewPlayerControl(keysEvents, controller.Player1DefaultKeyBinding)
		go calibration.Run(control)
	} else {
		screen, _ = NewPlayerSelectDialog(keysEvents, configurationChanel)
		render.Add(screen)
	}

	if DEBUG_MINIMAP {
		var debugMinimap func()
		debugMinimap = func() {

			/*if game.inProgress {
				navigation.SchedulePath(Zone{X:0, Y:0}, Zone{
					X: game.players[0].Unit.Tracker.xIndex,
					Y: game.players[0].Unit.Tracker.yIndex,
				}, game.players[0].Unit)
				logger.Printf("track from %d, %d to %d, %d \n", 0, 0, game.players[0].Unit.Tracker.xIndex, game.players[0].Unit.Tracker.yIndex)
				if len(navigation.NavData) > 0 {
					logger.Printf("last nav pos: ", navigation.NavData[0][len(navigation.NavData[0]) -1])
				}
			}*/
			logger.Println("dump minimap...")
			mmp, _ := location.Minimap(true, navigation.NavData)
			minimap.Printf("minimap for %d cycle \n\n", CycleID)
			for _, slice := range mmp {
				minimap.Printf("%s \n", slice)
			}
			minimap.Printf("\n\n\n")
			time.AfterFunc(time.Second*5, debugMinimap)
			navigation.NavData = navigation.NavData[0:0]
		}
		time.AfterFunc(time.Second*5, debugMinimap)
	}

	/*tank, _ := buildManager.Get("player-tank")
	startIndexX, startIndexY := 0, 0
	pos, _ := location.PosByIndex(startIndexX, startIndexY)
	tank.GetClBody().Moving(pos.X, pos.Y)
	x, y := tank.GetXY()
	w, h := tank.GetWH()
	tank.GetTracker().Manager = location
	tank.GetTracker().Update(x, y, w, h)
	newIndexX, newIndexY := tank.GetTracker().GetIndexes()
	log.Printf("unit size x, y  %f %f \n", location.setupUnitSize.X, location.setupUnitSize.Y)
	if newIndexX != startIndexX {
		log.Printf("x index broken %d %d \n", newIndexX, startIndexX)
	}
	if newIndexY != startIndexY {
		log.Printf("y index broken %d %d \n", newIndexY, startIndexY)
	}
	newPos, _, _, _ := location.NearestPos(x, y)
	if newPos.X != pos.X {
		log.Printf("x pos broken %f %f \n", newPos.X, pos.X)
	}
	if newPos.Y != pos.Y {
		log.Printf("y pos broken %f %f \n", newPos.Y, pos.Y)
	}
	tank.GetClBody().Moving(pos.X + 2, pos.Y + 2)
	x, y = tank.GetXY()
	tank.GetTracker().Manager = location
	tank.GetTracker().Update(x, y, w, h)
	newIndexX, newIndexY = tank.GetTracker().GetIndexes()
	log.Printf("x index  %d %d \n", newIndexX, startIndexX)
	log.Printf("y index  %d %d \n", newIndexY, startIndexY)
	tank.GetClBody().Moving(pos.X + 3, pos.Y + 3) //смещение по меньшей оси
	x, y = tank.GetXY()
	tank.GetTracker().Manager = location
	tank.GetTracker().Update(x, y, w, h)
	newIndexX, newIndexY = tank.GetTracker().GetIndexes()
	log.Printf("x index  %d %d \n", newIndexX, startIndexX)
	log.Printf("y index  %d %d \n", newIndexY, startIndexY)

	os.Exit(0)*/

	for {
		select {
		case configuration := <-configurationChanel:
			switch configuration.EType {
			case DIALOG_EVENT_PLAYER_SELECT:
				direct.Print("\033[?25l")
				screen.(*Dialog).Activate()

				payload := configuration.Payload.(*DialogInfo)
				for i := 0; i < payload.Value; i++ {
					//players
					playerControl, _ := controller.NewPlayerControl(keysEvents, controller.KeyboardBindingPool[i])
					player, _ := NewPlayer("Player"+strconv.Itoa(i+1), playerControl)
					player.CustomizeMap = &CustomizeMap{
						"gun":   direct.RED,
						"armor": direct.YELLOW,
						"track": direct.CYAN,
					}
					game.AddPlayer(player)
				}

				render.Remove(screen)
				screen.(*Dialog).Deactivate()

				direct.Clear()
				direct.Flush()

				go game.Run(scenario)
			case CALIBRATION_COMPLETE:
				direct.Clear()
				direct.Flush()
				err := calibration.End()
				if err != nil {
					direct.Printf("fail to save new config.json, %s", err)
				} else {
					direct.Printf("successfully save new config.json")
				}
				direct.Flush()
				return
			}
		case gameEvent := <-game.GetEventChanel():
			switch gameEvent.EType {
			case GAME_START:
				cycleTime = CYCLE
				endGame = false
				render.UI(&UIData{players: game.GetPlayers()})
				profileStart(profileMod, profileDelay)
			case GAME_END_LOSE:
				screen, _ = NewLoseScreen()
				cycleTime = SLOW_CYCLE
				endGame = true
				render.UI(nil)
				terminateEvent = time.After(10 * time.Second)
				resultScreen = time.After(200 * time.Millisecond)
			case GAME_END_WIN:
				screen, _ = NewWinScreen()
				cycleTime = SLOW_CYCLE
				endGame = true
				render.UI(nil)
				terminateEvent = time.After(10 * time.Second)
				resultScreen = time.After(200 * time.Millisecond)
			}
		case <-resultScreen:
			direct.Clear()
			direct.Flush()
			render.Add(screen)
		case <-terminateEvent:
			direct.Clear()
			direct.Printf("game seed is: %d \n", seed)
			direct.Flush()
			return
		case <-osSignal:
			direct.Clear()
			direct.Printf("game seed is: %d \n", seed)
			direct.Flush()
			return
		case timeEvent := <-timeEvents:
			timeLeft = timeEvent.Sub(timeCurrent)
			timeCurrent = timeEvent
			pipe.Execute(timeLeft)

			//todo dynamic cycleTime
			timeEvents = time.After(cycleTime)
			if CycleID == math.MaxInt64 {
				CycleID = 0
			} else {
				CycleID++
			}
		}
	}
}

func newTimer(duration time.Duration) <-chan time.Time {
	output := make(chan time.Time)

	go func(duration time.Duration, output chan time.Time) {
		events := time.After(duration)
		for {
			select {
			case timeLeft := <-events:
				output <- timeLeft
				events = time.After(duration)
			}
		}
	}(duration, output)

	return output
}

func profileStart(mode string, delay time.Duration) {
	//use the flags package to selectively enable profiling.

	do := func() {
		switch mode {
		case "cpu":
			profile.Start(profile.CPUProfile, profile.ProfilePath("./prof"))
		case "mem":
			profile.Start(profile.MemProfile, profile.ProfilePath("./prof"))
		case "mutex":
			profile.Start(profile.MutexProfile, profile.ProfilePath("./prof"))
		case "block":
			profile.Start(profile.BlockProfile, profile.ProfilePath("./prof"))
		case "all":
			profile.Start(profile.ProfilePath("./prof"))
		default:

		}
	}

	if delay != -1 {
		time.AfterFunc(delay, do)
	} else {
		do()
	}
}

func profileStop() {
	if profilerHandler != nil {
		profilerHandler.Stop()
	}
}
