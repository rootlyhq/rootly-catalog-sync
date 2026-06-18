package sync

import (
	"encoding/json"
	"os"
)

func SavePlan(plan *Plan, path string) error {
	data, err := json.MarshalIndent(plan, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

func LoadPlan(path string) (*Plan, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var plan Plan
	return &plan, json.Unmarshal(data, &plan)
}
