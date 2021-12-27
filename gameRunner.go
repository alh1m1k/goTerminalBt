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
	receiver.BlueprintManager.GameConfig = gameConfig
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

	receiver.setupPlayers()
	exitEvt := receiver.runGame()
	<-time.After(200 * time.Millisecond)
	exitEvt = receiver.resultScreen(exitEvt)
	if done != nil { //todo remove
		done <- exitEvt
	}
	return exitEvt
}

func (receiver *GameRunner) setupPlayers() {
	direct.Print("\033[?25l")

	var configurationChanel EventChanel = make(EventChanel) //todo remove
	screen, _ := NewPlayerSelectDialog(receiver.Keyboard, configurationChanel)
	receiver.Renderer.Add(screen)
	screen.Activate()

	for {
		select {
		case configuration := <-configurationChanel:
			switch configuration.EType {
			case DIALOG_EVENT_PLAYER_SELECT:
				payload := configuration.Payload.(*DialogInfo)
				for i := 0; i < payload.Value; i++ {
					playerControl, _ := controller.NewPlayerControl(receiver.Keyboard, controller.KeyboardBindingPool[i])
					player, _ := NewPlayer("Player"+strconv.Itoa(i+1), playerControl)
					player.CustomizeMap = &CustomizeMap{
						"gun":   direct.RED,
						"armor": direct.YELLOW,
						"track": direct.CYAN,
					}
					game.AddPlayer(player)
				}

				render.Remove(screen)
				screen.Deactivate()

				direct.Clear()
				direct.Flush()
				return
			}
		}
	}
}
func (receiver *GameRunner) runGame() (exitEvent Event) {
	go receiver.Game.Run(receiver.Scenario)
	for {
		select {
		case gameEvent := <-receiver.Game.GetEventChanel():
			switch gameEvent.EType {
			case GAME_START:
				receiver.Renderer.UI(&UIData{players: game.GetPlayers()})
				profileStart(profileMod, profileDelay)
			case GAME_END_WIN:
				fallthrough
			case GAME_END_LOSE:
				return gameEvent
			}
		}
	}
}
func (receiver *GameRunner) resultScreen(exitEvent Event) Event {
	var screen Screener
	direct.Clear()
	direct.Flush()
	switch exitEvent.EType {
	case GAME_END_LOSE:
		receiver.Renderer.UI(nil)
		screen, _ = NewLoseScreen()
	case GAME_END_WIN:
		receiver.Renderer.UI(nil)
		screen, _ = NewWinScreen()
	}
	receiver.Renderer.Add(screen)
	<-time.After(10 * time.Second)
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
