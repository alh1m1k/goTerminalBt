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
	"syscall"
	"time"
)

const CYCLE = 100 * time.Millisecond

const DEBUG = false
const DEBUG_SPAWN = false
const DEBUG_EVENT = false
const DEBUG_EXEC = false
const DEBUG_STATE = false
const DEBUG_NO_AI = false
const DEBUG_SHAKE = false
const DEBUG_IMMORTAL_PLAYER = true
const DEBUG_FREEZ_AI = false
const DEBUG_AI_PATH = false
const DEBUG_AI_BEHAVIOR = false
const DEBUG_FIRE_SOLUTION = false
const DEBUG_MINIMAP = false
const DEBUG_DISABLE_VISION = false
const DEBUG_SHUTDOWN = false
const DEBUG_OPPORTUNITY_FIRE = false

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
	gameConfig  *GameConfig
	game        *Game
	render      Renderer
	calibration *Calibration
	scenario    *Scenario
)

//flags
var (
	seed             int64
	wallCnt, tankCnt int
	calibrate        bool
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

	//Position
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

	//ai
	aibuilder, _ := NewAIControlBuilder(detector, location, navigation)

	//scenario
	if scenarioName == "random" {
		scenario, _ = NewRandomScenario(tankCnt, wallCnt)
	} else {
		scenario, err = GetScenario(scenarioName)
		if err != nil {
			logger.Print(err)
			log.Print(err)
			os.Exit(1)
		}
	}

	//game
	game, _ = NewGame(nil, spawner)
	game.Location = location
	game.EffectManager = pipe.EffectManager

	//runner
	runner, _ := NewGameRunner()
	runner.Keyboard = keysEvents
	runner.Game = game
	runner.Scenario = scenario
	runner.BlueprintManager = buildManager
	runner.BehaviorControlBuilder = aibuilder
	runner.SpawnManager = spawner
	runner.Renderer = render

	//time
	cycleTime := CYCLE
	var timeCurrent time.Time = time.Now()
	var timeLeft time.Duration
	timeEvents := time.After(cycleTime)

	direct.Clear()
	direct.MoveCursor(0, 0)
	direct.Flush()

	var finChanel EventChanel = make(EventChanel)
	if calibrate {
		calibration, _ = NewCalibration(updater, render, detector, location, finChanel)
		calibration.GameConfig = gameConfig
		control, _ := controller.NewPlayerControl(keysEvents, controller.Player1DefaultKeyBinding)
		go calibration.Run(control)
	} else {
		go runner.Run(game, scenario, finChanel)
	}

	if DEBUG_MINIMAP {
		var debugMinimap func()
		debugMinimap = func() {
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
	pos, _ := location.CoordinateByIndex(startIndexX, startIndexY)
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
	newPos, _, _, _ := location.NearestZoneByCoordinate(x, y)
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
		case fin := <-finChanel:
			switch fin.EType {
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
			case GAME_END_WIN:
				fallthrough
			case GAME_END_LOSE:
				direct.Clear()
				direct.Printf("game seed is: %d \n", seed)
				direct.Flush()
				return
			}
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
