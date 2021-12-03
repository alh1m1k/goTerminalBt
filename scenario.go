package main

import "math/rand"

const (
	SPAWN_REQUEST = iota + 700
)

var (
	SpawnReqEvent = Event{
		EType:   SPAWN_REQUEST,
		Object:  nil,
		Payload: nil,
	}
)

type SpawnRequest struct {
	Location  Point //maybe auto
	Team      int8
	Blueprint string
}

type ScenarioStateInfo struct {
	Declare []string
	Spawn   []*SpawnRequest
	Screen  *Screen
}

type Scenario struct {
	*State
	*ObservableObject
	declareBlueprint                   func(blueprint string)
	dropBlueprint                      func(blueprint string)
	declarationCleanup                 []string
	player1Blueprint, player2Blueprint string
}

func (receiver *Scenario) ApplyState(current *StateItem) error { /*
		//cleanup
		for _, blueprint := range receiver.declarationCleanup {
			receiver.dropBlueprint(blueprint)
		}
		receiver.declarationCleanup = receiver.declarationCleanup[0:0]*/

	//move on
	scenarioStateInfo := current.StateInfo.(*ScenarioStateInfo)
	for _, blueprint := range scenarioStateInfo.Declare {
		receiver.declareBlueprint(blueprint)
		receiver.declarationCleanup = append(receiver.declarationCleanup, blueprint)
	}

	for _, spawnOrder := range scenarioStateInfo.Spawn {
		receiver.Trigger(SpawnReqEvent, receiver, spawnOrder)
	}

	return nil
}

func (receiver *Scenario) DeclareBlueprint(fn func(blueprint string)) {
	receiver.declareBlueprint = fn //todo queue
}

func (receiver *Scenario) DropBlueprint(fn func(blueprint string)) {
	receiver.declareBlueprint = fn //todo queue
}

func NewScenario() (*Scenario, error) {
	instance := new(Scenario)
	instance.State, _ = NewState(instance)
	root, _ := NewStateItem(nil, nil)
	instance.State.root = root
	instance.State.Current = root
	instance.State.defaultPath = "/"
	instance.State.MoveTo("/")
	instance.ObservableObject, _ = NewObservableObject(make(EventChanel), instance)
	return instance, nil
}

func NewRandomScenario(tankCnt int, wallCnt int) (*Scenario, error) {
	scenario, err := NewScenario()
	if err != nil {
		return nil, err
	}
	spawn := make([]*SpawnRequest, 0, tankCnt+wallCnt)
	blList := []string{"tank", "tank", "tank", "tank-fast", "tank-fast", "tank-fast", "tank-heavy", "tank-sneaky", "tank-sneaky"}
	for i := 0; i < tankCnt; i++ {
		spawn = append(spawn, &SpawnRequest{
			Location:  PosAuto,
			Blueprint: blList[rand.Intn(len(blList))],
			Team:      1,
		})
	}
	blList = []string{"wall", "wall", "wall", "water"}
	for i := 0; i < wallCnt; i++ {
		spawn = append(spawn, &SpawnRequest{
			Location:  PosAuto,
			Blueprint: blList[rand.Intn(len(blList))],
			Team:      100,
		})
	}
	start, _ := NewStateItem(scenario.State.root, &ScenarioStateInfo{
		Declare: []string{
			"player-tank",
			"tank",
			"tank-fast",
			"tank-heavy",
			"tank-sneaky",
			"tank-base-explosion",
			"tank-base-projectile",
			"tank-base-projectile-he",
			"tank-base-projectile-rail",
			"tank-base-projectile-flak",
			"tank-special-projectile-smoke",
			"tank-special-smokescreen-1",
			"tank-special-smokescreen-2",
			"tank-base-projectile-fanout",
			"projectile-sharp",
			"tank-base-projectile-apocalypse",
			"projectile-sharp-apoc-start",
			"projectile-sharp-apoc-end",
			"opel",
			"gun",
			"water",
			"wall",
		},
		Spawn:  spawn,
		Screen: nil,
	})
	scenario.State.root.items["start"] = start

	return scenario, nil
}
