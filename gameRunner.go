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
	*Scenario
	*Game
	*GameConfig
	*BlueprintManager
	*BehaviorControlBuilder
	*SpawnManager
	Renderer
}

func (receiver *GameRunner) Init() {

}

func (receiver *GameRunner) Run(game *Game, scenario *Scenario, done EventChanel) (exitEvent Event) {
	receiver.Game, receiver.Scenario = game, scenario

	receiver.BlueprintManager.AddLoaderPackage(NewJsonPackage())
	receiver.BlueprintManager.GameConfig = receiver.GameConfig
	receiver.BlueprintManager.EventChanel = receiver.SpawnManager.UnitEventChanel //remove from builder
	receiver.BlueprintManager.AddLoader("ai", func(get LoaderGetter, eCollector *LoadErrors, payload []byte) interface{} {
		ai, _ := receiver.BehaviorControlBuilder.Build()
		return ai
	})

	scenario.DeclareBlueprint(func(blueprint string) {
		builder, _ := receiver.BlueprintManager.CreateBuilder(blueprint)
		if builder == nil { //may cause error on success
			panic("builder " + blueprint + " not found")
		} else {
			receiver.SpawnManager.AddBuilder(blueprint, builder)
		}
		object, _ := receiver.BlueprintManager.Get(blueprint)
		if projectile, ok := object.(*Projectile); ok {
			if err := receiver.BehaviorControlBuilder.RegisterProjectile(projectile); err != nil {
				logger.Println(err)
			}
		}
	})

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

	screen, _ := NewSetupSizeDialog(receiver.GameConfig.Box, receiver.Keyboard, configurationChanel)
	receiver.Renderer.Add(screen)
	screen.Activate()

	for {
		select {
		case configuration := <-configurationChanel:
			switch configuration.EType {
			case DIALOG_EVENT_SETUP_SIZE:
				screen.Deactivate()
				receiver.Renderer.Remove(screen)
				direct.Clear()
				direct.Flush() //todo may cause bug must be in render
				return
			}
		}
	}
}

func (receiver *GameRunner) setupPlayers() {

	var configurationChanel EventChanel = make(EventChanel) //todo remove
	screen, _ := NewPlayerSelectDialog(receiver.Keyboard, configurationChanel)
	receiver.Renderer.Add(screen)
	screen.Activate()

	for {
		select {
		case configuration := <-configurationChanel:
			switch configuration.EType {
			case DIALOG_EVENT_PLAYER_SELECT:
				screen.Deactivate()
				payload := configuration.Payload.(*DialogInfo)
				keyboardRepeater, _ := NewKeyboardRepeater(receiver.Keyboard)
				for i := 0; i < payload.Value; i++ {
					playerControl, _ := controller.NewPlayerControl(keyboardRepeater.Subscribe(), controller.KeyboardBindingPool[i])
					player, _ := NewPlayer("Player"+strconv.Itoa(i+1), playerControl)
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
