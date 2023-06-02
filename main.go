package main

import (
	"GoConsoleBT/collider"
	"GoConsoleBT/controller"
	"GoConsoleBT/output"
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

/**
* Go BattleTanks v0.3
* @author Pyadukhov Roman
 */

const CYCLE = 100 * time.Millisecond

const DEBUG = false
const DEBUG_SPAWN = false
const DEBUG_EVENT = false
const DEBUG_EXEC = false
const DEBUG_STATE = false
const DEBUG_NO_AI = false
const DEBUG_SHAKE = false
const DEBUG_IMMORTAL_PLAYER = false
const DEBUG_FREEZ_AI = false
const DEBUG_AI_PATH = false
const DEBUG_AI_BEHAVIOR = false
const DEBUG_FIRE_SOLUTION = false
const DEBUG_MINIMAP = false
const DEBUG_DISABLE_VISION = false
const DEBUG_SHUTDOWN = false
const DEBUG_OPPORTUNITY_FIRE = false
const DEBUG_DISABLE_UI = false
const DEBUG_DISARM_AI = false
const DEBUG_SHOW_ID = false
const DEBUG_SHOW_AI_BEHAVIOR = false
const DEBUG_FREE_SPACES = false
const DEBUG_SPAWN_POINT_STATUS = false

var (
	buf, bufErr     = os.OpenFile("log.txt", os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 644)
	logger          = log.New(buf, "logger: ", log.Lshortfile)
	profilerHandler interface {
		Stop()
	}
	CycleID int64 = 0

	gameConfig   *GameConfig
	game         *Game
	render       Renderer
	calibration  *Calibration
	scenario     *Scenario
	aibuilder    *BehaviorControlBuilder
	sound        *SoundManager
	buildManager *BlueprintManager
	ui           *UI
	Require      RequireFunc
	Info         InfoFunc

	//flags
	seed                         int64
	wallCnt, tankCnt, limitMaxAi int
	calibrate                    bool
	profileMod                   string
	scenarioName                 string
	profileDelay                 time.Duration
	withColor, withSound         bool
	simplifyAi                   bool
	osSignal                     chan os.Signal
)

func init() {
	dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		log.Fatal(err)
	}
	if bufErr != nil && DEBUG {
		panic(bufErr)
	}
	profile.ProfilePath(dir)
	flag.Int64Var(&seed, "seed", time.Now().UnixNano(), "random generator seed")
	flag.StringVar(&scenarioName, "scenario", "random", "run scenario")
	flag.IntVar(&tankCnt, "tankCnt", 25, "for random scenario")
	flag.IntVar(&wallCnt, "wallCnt", 80, "for random scenario")
	flag.IntVar(&limitMaxAi, "limit.maxAiUnit", 10, "for random scenario, 0 means no limit")
	flag.BoolVar(&calibrate, "calibrate", false, "terminal calibration mode")
	flag.StringVar(&profileMod, "profile.mode", "", "enable profiling mode, one of [cpu, mem, mutex, block, all]")
	flag.DurationVar(&profileDelay, "profile.delay", -1, "delay of starting profile, after game start. -1 means no delay")
	flag.BoolVar(&withColor, "withColor", false, "enable color mode (3bit mode (8 color))")
	flag.BoolVar(&withSound, "withSound", false, "enable sound mode (the sounds will be played on the machine where the game is running)")
	flag.BoolVar(&simplifyAi, "simplifyAi", false, "disable ai behaviors")

	osSignal = make(chan os.Signal, 1)

	//cause problem on debug
	signal.Notify(osSignal, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT, syscall.SIGKILL)

	output.DEBUG = DEBUG
	controller.DEBUG_DISARM_AI = DEBUG_DISARM_AI
}

func main() {
	var err error

	defer buf.Close()

	//EffectAnimDisappear("stealth/tank/left/tank", 16, 42)
	//EffectAnimInterference("napalm/persist/smokeGrow", 6, 0.3)

	//EffectVerFlip(bytes.NewReader([]byte("123\n456\n789")), os.Stdout)
	/*	EffectAnimNormalizeNewLine("stealth/tank/right/tank", 17)
		EffectAnimVerFlip("stealth/tank/right/tank", 17)*/

	/*	EffectAnimNormalizeNewLine("flak/left/flak", 10)
		EffectAnimVerFlip("flak/left/flak", 10)*/

	//os.Exit(0)
	flag.Parse()

	rand.Seed(seed)

	gameConfig, err = loadConfig()
	if err != nil {
		if !calibrate {
			gameConfig, _ = NewDefaultGameConfig()
			saveConfig(gameConfig)
			log.Println("no config found, default created. Run --calibrate if you want custom config. Restart game.")
			return
		} else {
			gameConfig, _ = NewDefaultGameConfig()
		}
	}
	gameConfig.disableCustomization = !withColor

	//input
	keysEvents, err := keyboard.GetKeys(1)
	if err != nil {
		panic(err)
	}
	repeater, _ := NewKeyboardRepeater(keysEvents)
	closingEvents := repeater.Subscribe()

	//closing
	defer func() {
		_ = keyboard.Close()
		profileStop()
		render.Free()
		buf.Sync()
		buf.Close()
		if !DEBUG {
			direct.Clear()
			direct.Printf("game seed is: %d \n", seed)
			direct.Flush()
		}
	}()

	//start pipeline
	pipe, _ := NewGPipeline()

	//animation
	animator, _ := getAnimationManager()
	pipe.AnimationManager = animator

	//render
	render, _ = NewRenderZIndex(100)
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

	//scenario
	if scenarioName == "random" {
		scenario, _ = NewRandomScenario(tankCnt, wallCnt, limitMaxAi)
	} else {
		scenario, err = GetScenario(scenarioName)
		if err != nil {
			logger.Print(err)
			log.Print(err)
			os.Exit(1)
		}
	}

	//Position
	var size Box
	if scenario.Location == EmptyLocation {
		size = gameConfig.Box
		size.Y += 3
		size.H -= 3 //respect UI, todo try to impl something better
	} else {
		size = scenario.Location
		size.Y += 3 //expect that scenario know about UI offset
	}
	location, _ := NewLocation(size.Point, size.Size)
	detector.Add(location)
	pipe.Location = location

	//spawner
	spawner, _ := NewSpawner(updater, render, detector, location, vision, gameConfig)
	pipe.SpawnManager = spawner

	navigation, _ := NewNavigation(location, detector)
	pipe.Navigation = navigation

	//builder
	buildManager, _ = NewBlueprintManager()
	Require = func(blueprint string) error {
		_, err := buildManager.Get(blueprint)
		return err
	}
	Info = func(blueprint string) (BlueprintInfo, error) {
		return buildManager.Info(blueprint)
	}

	//ai
	if !simplifyAi {
		aibuilder, _ = NewAIControlBuilder(detector, location, navigation)
	}

	if withSound {
		sound, _ = NewSoundManager()
	}

	//game
	game, _ = NewGame(nil, spawner)
	game.Location = location
	game.EffectManager = pipe.EffectManager
	game.SoundManager = sound

	//ui
	if !DEBUG_DISABLE_UI {
		ui, _ = NewDefaultUI()
		pipe.UI = ui
	}

	//runner
	runner, _ := NewGameRunner()
	runner.Keyboard = keysEvents
	runner.KeyboardRepeater = repeater
	runner.Game = game
	runner.GameConfig = gameConfig
	runner.Scenario = scenario
	runner.BlueprintManager = buildManager
	runner.BehaviorControlBuilder = aibuilder
	runner.SpawnManager = spawner
	runner.Renderer = render
	runner.SoundManager = sound
	runner.UI = ui

	//time
	cycleTime := CYCLE
	var timeCurrent time.Time = time.Now()
	var timeLeft time.Duration
	cycleTimer := time.NewTimer(cycleTime)

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

	if DEBUG_FREE_SPACES {
		var debugFreeSpaces func()
		debugFreeSpaces = func() {
			logger.Printf("cycleId: %d, free spaces %d", CycleID, location.zonesLeft)

		}
		time.AfterFunc(time.Second*5, debugFreeSpaces)
	}

	if profileMod != "" {
		profileStart(profileMod, profileDelay)
	}

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
				return
			}
		case <-osSignal:
			return
		case event := <-closingEvents:
			if event.Key == keyboard.KeyCtrlC {
				return
			}
		case timeEvent := <-cycleTimer.C:
			timeLeft = timeEvent.Sub(timeCurrent)
			timeCurrent = timeEvent
			pipe.Execute(timeLeft)
			cycleTime = CYCLE - time.Now().Sub(timeCurrent)
			if cycleTime <= time.Millisecond {
				cycleTime = time.Millisecond
			}
			cycleTimer.Reset(cycleTime)
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
		logger.Print("start profile")
		switch mode {
		case "cpu":
			profilerHandler = profile.Start(func(p *profile.Profile) {
				profile.CPUProfile(p)
				profile.NoShutdownHook(p)
			}, profile.ProfilePath("./prof"))
		case "mem":
			profilerHandler = profile.Start(func(p *profile.Profile) {
				profile.CPUProfile(p)
				profile.NoShutdownHook(p)
			}, profile.ProfilePath("./prof"))
		case "mutex":
			profilerHandler = profile.Start(func(p *profile.Profile) {
				profile.MutexProfile(p)
				profile.NoShutdownHook(p)
			}, profile.ProfilePath("./prof"))
		case "block":
			profilerHandler = profile.Start(func(p *profile.Profile) {
				profile.BlockProfile(p)
				profile.NoShutdownHook(p)
			}, profile.ProfilePath("./prof"))
		case "all":
			profilerHandler = profile.Start(func(p *profile.Profile) {
				profile.CPUProfile(p)
				profile.MutexProfile(p)
				profile.BlockProfile(p)
				profile.NoShutdownHook(p)
			}, profile.ProfilePath("./prof"))
		default:
			logger.Print("wrong profile type")
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
		logger.Print("stop profile")
		profilerHandler.Stop()
	}
}
