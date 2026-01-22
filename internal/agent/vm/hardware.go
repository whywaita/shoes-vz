package vm

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"

	"github.com/Code-Hex/vz/v3"
)

// HardwareModelJSON represents the JSON structure of HardwareModel.json
type HardwareModelJSON struct {
	HardwareModel string `json:"hardwareModel"`
}

// LoadHardwareModel loads a hardware model from a JSON file
// The JSON file should contain a base64-encoded hardware model in the "hardwareModel" field
func LoadHardwareModel(path string) (*vz.MacHardwareModel, error) {
	// Read JSON file
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read hardware model file: %w", err)
	}

	// Parse JSON
	var hwModel HardwareModelJSON
	if err := json.Unmarshal(data, &hwModel); err != nil {
		return nil, fmt.Errorf("failed to parse hardware model JSON: %w", err)
	}

	// Decode base64
	hwData, err := base64.StdEncoding.DecodeString(hwModel.HardwareModel)
	if err != nil {
		return nil, fmt.Errorf("failed to decode hardware model: %w", err)
	}

	// Create hardware model from binary data
	model, err := vz.NewMacHardwareModelWithData(hwData)
	if err != nil {
		return nil, fmt.Errorf("failed to create hardware model: %w", err)
	}

	return model, nil
}
