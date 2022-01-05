package main

import (
	"encoding/json"
	"errors"
	"math/rand"
)

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
	Position  Point //maybe auto
	Location  Zone  //maybe auto
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
			Position:  PosAuto,
			Location:  ZoneAuto,
			Blueprint: blList[rand.Intn(len(blList))],
			Team:      1,
		})
	}
	blList = []string{"wall", "wall", "wall", "water", "forest"}
	for i := 0; i < wallCnt; i++ {
		spawn = append(spawn, &SpawnRequest{
			Position:  PosAuto,
			Location:  ZoneAuto,
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
			"effect-onsight",
			"effect-offsight",
			"opel",
			"gun",
			"water",
			"wall",
			"forest",
		},
		Spawn:  spawn,
		Screen: nil,
	})
	scenario.State.root.items["start"] = start

	return scenario, nil
}

func NewFileScenario(filepath string) (*Scenario, error) {
	state, err := GetScenarioState(filepath)
	if err != nil {
		return nil, err
	}

	instance := new(Scenario)
	instance.State = state
	instance.State.Owner = instance
	instance.ObservableObject, _ = NewObservableObject(make(EventChanel), instance)

	return instance, nil
}

func GetScenario(name string) (*Scenario, error) {
	switch name {
	default:
		return NewFileScenario(name)
	}
}

func GetScenarioState(id string) (*State, error) { //todo refactor
	//<-time.After(time.Second * 5)
	return LoadScenario(id, func(m map[string]interface{}) (interface{}, error) {
		var ssi *ScenarioStateInfo = &ScenarioStateInfo{
			Declare: make([]string, 0),
			Spawn:   make([]*SpawnRequest, 0),
			Screen:  nil,
		}

		if declare, ok := m["declare"]; ok {
			//todo refactor this shit
			for _, decl := range declare.([]interface{}) {
				ssi.Declare = append(ssi.Declare, decl.(string))
			}
		}
		if _, ok := m["screen"]; ok {

		}
		if spawn, ok := m["spawn"]; ok {
			//todo refactor this shit
			for _, spawnItem := range spawn.([]interface{}) {
				spr := SpawnRequest{}
				spr.Position = PosAuto
				spr.Location = ZoneAuto
				for key, value := range spawnItem.(map[string]interface{}) {
					switch key {
					case "team":
						spr.Team = int8(value.(float64))
					case "blueprint":
						spr.Blueprint = value.(string)
					case "position":
						loc := value.(map[string]interface{})
						spr.Position = Point{}
						spr.Position.X = loc["X"].(float64)
						spr.Position.Y = loc["Y"].(float64)
					case "location":
						loc := value.(map[string]interface{})
						spr.Location = Zone{}
						spr.Location.X = int(loc["X"].(float64))
						spr.Location.Y = int(loc["Y"].(float64))
					}
				}
				ssi.Spawn = append(ssi.Spawn, &spr)
			}
		}

		return ssi, nil
	})
}

func LoadScenario(id string, builder SateInfoBuilder) (*State, error) { //todo refactor
	if state, ok := states[id]; ok {
		return state.Copy(), nil
	}
	buffer, err := loadScenario(id)
	if err != nil {
		return nil, err
	}
	stateRead := stateRead{}
	err = json.Unmarshal(buffer, &stateRead)
	if err != nil {
		return nil, err
	}
	if len(stateRead.Items) == 0 {
		return nil, errors.New("load empty state")
	}

	state, _ := NewState(nil)
	root := state.root

	recursiveCreateState(root, stateRead.Items, builder)

	if stateRead.Default != "" {
		err = state.MoveTo(stateRead.Default)
		if err != nil {
			return nil, err
		}
		state.defaultPath = stateRead.Default
	}

	states[id] = state
	return state.Copy(), nil
}
