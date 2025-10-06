package miner

import (
	"encoding/json"
	"fmt"
	"os"
)

type PersistentState struct {
	PredictionResults map[string]map[string]int `json:"prediction_results"`
}

func (p *PersistentState) Save(opt Options) error {
	if opt.PersistentFile == "" {
		return nil
	}

	fd, err := os.Create(opt.PersistentFile)
	if err != nil {
		return err
	}
	defer func() {
		_ = fd.Close()
	}()

	encoder := json.NewEncoder(fd)
	encoder.SetIndent("", "\t")
	if err := encoder.Encode(p); err != nil {
		return err
	}
	if err := fd.Sync(); err != nil {
		return err
	}
	return nil
}

func LoadPersistentState(opt Options) *PersistentState {
	if opt.PersistentFile == "" {
		return freshPersistentState()
	}

	var persistentState PersistentState
	fd, err := os.Open(opt.PersistentFile)
	if err != nil {
		fmt.Println("Failed to open persistent state file", err)
		return freshPersistentState()
	}
	defer func() {
		_ = fd.Close()
	}()

	if err := json.NewDecoder(fd).Decode(&persistentState); err != nil {
		fmt.Println("Failed to unmarshal persistent state", err)
		return freshPersistentState()
	}

	if persistentState.PredictionResults == nil {
		persistentState.PredictionResults = map[string]map[string]int{}
	}

	return &persistentState
}

func freshPersistentState() *PersistentState {
	return &PersistentState{
		PredictionResults: map[string]map[string]int{},
	}
}
