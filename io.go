package main

import (
	"GoConsoleBT/controller"
	"encoding/json"
	"github.com/buger/jsonparser"
	"os"
)

const spritePath = "./sprite/"
const statePath = "./state/"
const scenarioPath = "./scenario/"

func loadSprite(filename string) ([]byte, error) {
	return os.ReadFile(spritePath + filename)
}

func loadState(filename string) ([]byte, error) {
	return os.ReadFile(statePath + filename + ".json")
}

func loadScenario(filename string) ([]byte, error) {
	return os.ReadFile(scenarioPath + filename + ".json")
}

func saveConfig(config *GameConfig) (int, error) {
	payload, err := json.Marshal(config)
	if err != nil {
		return 0, err
	}
	return len(payload), os.WriteFile("config.json", payload, 644)
}

func loadConfig() (*GameConfig, error) {
	payload, err := os.ReadFile("config.json")
	if err != nil {
		return nil, err
	}
	config := new(GameConfig)
	err = json.Unmarshal(payload, config)
	kb, dType, _, _ := jsonparser.Get(payload, "keyBindings")
	switch dType {
	case jsonparser.Array:
		idx := 0
		jsonparser.ArrayEach(kb, func(value []byte, dataType jsonparser.ValueType, offset int, err error) {
			bind := &controller.KeyBind{}
			json.Unmarshal(value, bind)
			config.KeyBindings[idx] = *bind
			idx++
		})
	}
	return config, err
}
