package session

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type Storage struct {
	configPath string
}

type StorageData struct {
	Instances []*Instance `json:"instances"`
}

func NewStorage() (*Storage, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	configDir := filepath.Join(homeDir, ".config", "claude-session-manager")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create config directory: %w", err)
	}

	return &Storage{
		configPath: filepath.Join(configDir, "sessions.json"),
	}, nil
}

func (s *Storage) Load() ([]*Instance, error) {
	data, err := os.ReadFile(s.configPath)
	if os.IsNotExist(err) {
		return []*Instance{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var storageData StorageData
	if err := json.Unmarshal(data, &storageData); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Update status for all instances
	for _, instance := range storageData.Instances {
		instance.UpdateStatus()
	}

	return storageData.Instances, nil
}

func (s *Storage) Save(instances []*Instance) error {
	storageData := StorageData{
		Instances: instances,
	}

	data, err := json.MarshalIndent(storageData, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(s.configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

func (s *Storage) AddInstance(instance *Instance) error {
	instances, err := s.Load()
	if err != nil {
		return err
	}

	// Check for duplicate names
	for _, inst := range instances {
		if inst.Name == instance.Name {
			return fmt.Errorf("instance with name '%s' already exists", instance.Name)
		}
	}

	instances = append(instances, instance)
	return s.Save(instances)
}

func (s *Storage) RemoveInstance(id string) error {
	instances, err := s.Load()
	if err != nil {
		return err
	}

	newInstances := make([]*Instance, 0, len(instances))
	found := false
	for _, inst := range instances {
		if inst.ID == id {
			found = true
			// Stop the instance if running
			inst.Stop()
			continue
		}
		newInstances = append(newInstances, inst)
	}

	if !found {
		return fmt.Errorf("instance not found")
	}

	return s.Save(newInstances)
}

func (s *Storage) UpdateInstance(instance *Instance) error {
	instances, err := s.Load()
	if err != nil {
		return err
	}

	for i, inst := range instances {
		if inst.ID == instance.ID {
			instances[i] = instance
			return s.Save(instances)
		}
	}

	return fmt.Errorf("instance not found")
}

func (s *Storage) GetInstance(id string) (*Instance, error) {
	instances, err := s.Load()
	if err != nil {
		return nil, err
	}

	for _, inst := range instances {
		if inst.ID == id {
			return inst, nil
		}
	}

	return nil, fmt.Errorf("instance not found")
}

func (s *Storage) GetInstanceByName(name string) (*Instance, error) {
	instances, err := s.Load()
	if err != nil {
		return nil, err
	}

	for _, inst := range instances {
		if inst.Name == name {
			return inst, nil
		}
	}

	return nil, fmt.Errorf("instance not found")
}

// SaveAll saves all instances (preserving order) - used for reordering
func (s *Storage) SaveAll(instances []*Instance) error {
	return s.Save(instances)
}
