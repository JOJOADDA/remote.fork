package fileagenttasks

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
)

const schemaVersion = 1

type schemaState struct {
	AgentTasks int `json:"agentTasks"`
}

func migrate(root string) error {
	if err := os.MkdirAll(root, 0o755); err != nil {
		return err
	}
	tasksRoot := filepath.Join(root, "agent-tasks")
	if err := os.MkdirAll(tasksRoot, 0o755); err != nil {
		return err
	}
	path := filepath.Join(root, "agent-tasks-schema.json")
	state := schemaState{}
	data, err := os.ReadFile(path)
	if err == nil {
		if err := json.Unmarshal(data, &state); err != nil {
			return err
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if state.AgentTasks >= schemaVersion {
		return nil
	}
	state.AgentTasks = schemaVersion
	encoded, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	temp := path + ".tmp"
	if err := os.WriteFile(temp, encoded, 0o644); err != nil {
		return err
	}
	return os.Rename(temp, path)
}
