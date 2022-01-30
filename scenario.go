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
	EmptyLocation = Box{
		Point{},
		Size{},
	}
)

type SpawnRequest struct {
	Position  Point //maybe auto
	Location  Zone  //maybe auto
	Team      int8
	Blueprint string
	Count     int
}

type ScenarioLimits struct {
	AiUnits int64
}

type ScenarioStateInfo struct {
	Declare                            []string
	Spawn                              []*SpawnRequest
	Screen                             *Screen
	Location                           Box
	player1Blueprint, player2Blueprint string
	limits                             ScenarioLimits
}

type Scenario struct {
	*State
	*ObservableObject
	declareBlueprint                   func(blueprint string)
	dropBlueprint                      func(blueprint string)
	declarationCleanup                 []string
	player1Blueprint, player2Blueprint string
	limits                             ScenarioLimits
	Location                           Box
}

func (receiver *Scenario) ApplyState(current *StateItem) error { /*
		//cleanup
		for _, blueprint := range receiver.declarationCleanup {
			receiver.dropBlueprint(blueprint)
		}
		receiver.declarationCleanup = receiver.declarationCleanup[0:0]*/

	//move on
	scenarioStateInfo := current.StateInfo.(*ScenarioStateInfo)

	receiver.Location = scenarioStateInfo.Location
	receiver.player1Blueprint = scenarioStateInfo.player1Blueprint
	receiver.player2Blueprint = scenarioStateInfo.player2Blueprint
	receiver.limits = scenarioStateInfo.limits

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

func NewRandomScenario(tankCnt int, wallCnt int) (scenario *Scenario, err error) {
	scenario, err = NewScenario()
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
			Team:      1,
		})
	}
	start, _ := NewStateItem(scenario.State.root, &ScenarioStateInfo{
		player1Blueprint: "player-tank",
		player2Blueprint: "player-tank",
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
			"tank-base-projectile-fanout",
			"tank-base-projectile-apocalypse",
			"tank-special-projectile-napalm",
			"tank-base-projectile-apocalypse-napalm",
			"effect-onsight",
			"effect-offsight",
			"opel",
			"gun",
			"water",
			"wall",
			"forest",
			"player-base",
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

	if instance.State != nil {
		stateItem, _, err := instance.State.find("/start") //todo remove it
		if err == nil {
			info := stateItem.StateInfo.(*ScenarioStateInfo)
			instance.Location = info.Location
			instance.player1Blueprint = info.player1Blueprint
			instance.player2Blueprint = info.player2Blueprint
		}
	}

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

		if player1Blueprint, ok := m["player1Blueprint"]; ok {
			//todo refactor this shit
			ssi.player1Blueprint = player1Blueprint.(string)
		}
		if player2Blueprint, ok := m["player2Blueprint"]; ok {
			//todo refactor this shit
			ssi.player2Blueprint = player2Blueprint.(string)
		}
		if location, ok := m["location"]; ok {
			//todo refactor this shit
			locationMap := location.(map[string]interface{})
			ssi.Location.X = locationMap["x"].(float64)
			ssi.Location.Y = locationMap["y"].(float64)
			ssi.Location.W = locationMap["w"].(float64)
			ssi.Location.H = locationMap["h"].(float64)
		}
		if limits, ok := m["limits"]; ok {
			limitsMap := limits.(map[string]interface{})
			ssi.limits.AiUnits = int64(limitsMap["aiUnit"].(float64))
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
					case "count":
						spr.Count = int(value.(float64))
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
