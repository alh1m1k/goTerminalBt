package main

import (
	"encoding/json"
	"os"
)

const spritePath 	= "./sprite/"
const statePath 	= "./state/"

func loadSprite(filename string) ([]byte, error)   {
	return os.ReadFile(spritePath + filename)
}

func loadState(filename string) ([]byte, error)   {
	return os.ReadFile(statePath + filename + ".json")
}

func saveConfig(config *GameConfig) (int, error)   {
	payload, err := json.Marshal(config)
	if err != nil {
		return 0, err
	}
	return len(payload), os.WriteFile( "config.json", payload, 644)
}

func loadConfig() (*GameConfig, error)   {
	payload, err := os.ReadFile( "config.json")
	if err != nil {
		return nil, err
	}
	config := new(GameConfig)
	json.Unmarshal(payload, config)
	return config, json.Unmarshal(payload, config)
}
