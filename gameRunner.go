package main

import (
	"GoConsoleBT/controller"
	direct "github.com/buger/goterm"
	"github.com/eiannone/keyboard"
	"strconv"
	"time"
)

type GameRunner struct {
	Keyboard <-chan keyboard.KeyEvent
	*KeyboardRepeater
	*Scenario
	*Game
	*GameConfig
	*BlueprintManager
	*BehaviorControlBuilder
	*SpawnManager
	*SoundManager
	Renderer
}

func (receiver *GameRunner) Init() {

}

func (receiver *GameRunner) Run(game *Game, scenario *Scenario, done EventChanel) (exitEvent Event) {
	receiver.Game, receiver.Scenario = game, scenario

	if receiver.KeyboardRepeater == nil {
		receiver.KeyboardRepeater, _ = NewKeyboardRepeater(receiver.Keyboard)
	}

	receiver.BlueprintManager.AddLoaderPackage(NewJsonPackage())
	receiver.BlueprintManager.GameConfig = receiver.GameConfig
	receiver.BlueprintManager.EventChanel = receiver.SpawnManager.UnitEventChanel //remove from builder
	if receiver.BehaviorControlBuilder != nil {
		receiver.BlueprintManager.AddLoader("ai", func(get LoaderGetter, eCollector *LoadErrors, payload []byte) interface{} {
			ai, _ := receiver.BehaviorControlBuilder.Build()
			return ai
		})
	}

	scenario.DeclareBlueprint(func(blueprint string) {
		builder, _ := receiver.BlueprintManager.CreateBuilder(blueprint)
		if builder == nil { //may cause error on success
			panic("builder " + blueprint + " not found")
		} else {
			receiver.SpawnManager.AddBuilder(blueprint, builder)
		}
		if receiver.BehaviorControlBuilder != nil {
			object, _ := receiver.BlueprintManager.Get(blueprint)
			if projectile, ok := object.(*Projectile); ok {
				if err := receiver.BehaviorControlBuilder.RegisterProjectile(projectile); err != nil {
					logger.Println(err)
				}
			}
		}
	})

	//temporal
	if receiver.SoundManager != nil {
		err := receiver.SoundManager.Register("main", "./sounds/main.mp3", false)
		if err != nil {
			logger.Println(err)
		}
		for key, path := range map[string]string{
			"fire":      "./sounds/fire.mp3",
			"explosion": "./sounds/explosion.mp3",
			"damage":    "./sounds/damage.mp3",
		} {
			err = receiver.SoundManager.Register(key, path, true)
			if err != nil {
				logger.Println(err)
			}
		}
	}

	receiver.clear()
	receiver.wait(200 * time.Millisecond)
	receiver.setupSize()
	receiver.clear()
	receiver.wait(200 * time.Millisecond)
	receiver.setupPlayers()
	receiver.clear()
	exitEvt := receiver.runGame()
	receiver.wait(200 * time.Millisecond)
	receiver.clear()
	exitEvt = receiver.resultScreen(exitEvt) //warn for async render

	receiver.SpawnManager.Free()

	if done != nil { //todo remove
		done <- exitEvt
	}

	return exitEvt
}

func (receiver *GameRunner) setupSize() {
	var configurationChanel EventChanel = make(EventChanel) //todo remove

	keyboard := receiver.KeyboardRepeater.Subscribe()
	screen, _ := NewSetupSizeDialog(receiver.GameConfig.Box, keyboard, configurationChanel)
	receiver.Renderer.Add(screen)
	screen.Activate()

	for {
		select {
		case configuration := <-configurationChanel:
			switch configuration.EType {
			case DIALOG_EVENT_SETUP_SIZE:
				screen.Deactivate()
				receiver.Renderer.Remove(screen)
				receiver.KeyboardRepeater.Unsubscribe(keyboard)
				return
			}
		}
	}
}

func (receiver *GameRunner) setupPlayers() {

	var configurationChanel EventChanel = make(EventChanel) //todo remove
	keyboard := receiver.KeyboardRepeater.Subscribe()
	screen, _ := NewPlayerSelectDialog(keyboard, configurationChanel)
	receiver.Renderer.Add(screen)
	screen.Activate()

	for {
		select {
		case configuration := <-configurationChanel:
			switch configuration.EType {
			case DIALOG_EVENT_PLAYER_SELECT:
				screen.Deactivate()
				receiver.KeyboardRepeater.Unsubscribe(keyboard)
				payload := configuration.Payload.(*DialogInfo)
				for i := 0; i < payload.Value; i++ {
					pKeyboard := receiver.KeyboardRepeater.Subscribe()
					playerControl, _ := controller.NewPlayerControl(pKeyboard, receiver.GameConfig.KeyBindings[i])
					player, _ := NewPlayer("Player"+strconv.Itoa(i+1), playerControl)
					player.Keyboard = pKeyboard
					player.CustomizeMap = &CustomizeMap{
						"gun":   direct.RED,
						"armor": direct.YELLOW,
						"track": direct.CYAN,
					}
					game.AddPlayer(player)
				}
				receiver.Renderer.Remove(screen)
				return
			}
		}
	}
}

func (receiver *GameRunner) wait(duration time.Duration) {
	<-time.After(duration)
}

func (receiver *GameRunner) clear() {
	/*	direct.Clear()
		direct.Flush()*/
}

func (receiver *GameRunner) runGame() (exitEvent Event) {
	go receiver.Game.Run(receiver.Scenario)
	for {
		select {
		case gameEvent := <-receiver.Game.GetEventChanel():
			switch gameEvent.EType {
			case GAME_START:
				receiver.Renderer.UI(&UIData{players: game.GetPlayers()})
			case GAME_END_WIN:
				fallthrough
			case GAME_END_LOSE:
				receiver.Renderer.UI(nil)
				for _, player := range receiver.players {
					receiver.KeyboardRepeater.Unsubscribe(player.Keyboard)
				}
				if DEBUG_SHUTDOWN {
					logger.Println("receive GAME_END event")
				}
				return gameEvent
			}
		}
	}
}
func (receiver *GameRunner) resultScreen(exitEvent Event) Event {
	var screen Screener
	switch exitEvent.EType {
	case GAME_END_LOSE:
		screen, _ = NewLoseScreen()
	case GAME_END_WIN:
		screen, _ = NewWinScreen()
	}
	receiver.Renderer.Add(screen)
	receiver.wait(10 * time.Second)
	receiver.Renderer.Remove(screen)
	return exitEvent
}

func (receiver *GameRunner) lookupScreen(name string) (*Screen, error) {
	switch name {

	}
	return nil, nil
}

func NewGameRunner() (*GameRunner, error) {
	return &GameRunner{}, nil
}
