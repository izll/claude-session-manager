package session

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

type Storage struct {
	configPath string
}

// Group represents a session group for organizing sessions
type Group struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Collapsed    bool   `json:"collapsed"`
	Color        string `json:"color,omitempty"`          // Group name color
	BgColor      string `json:"bg_color,omitempty"`       // Background color
	FullRowColor bool   `json:"full_row_color,omitempty"` // Extend background to full row
}

// Settings stores UI preferences
type Settings struct {
	CompactList     bool `json:"compact_list"`
	HideStatusLines bool `json:"hide_status_lines"`
}

type StorageData struct {
	Instances []*Instance `json:"instances"`
	Groups    []*Group    `json:"groups,omitempty"`
	Settings  *Settings   `json:"settings,omitempty"`
}

func NewStorage() (*Storage, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	configDir := filepath.Join(homeDir, ".config", "agent-session-manager")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create config directory: %w", err)
	}

	return &Storage{
		configPath: filepath.Join(configDir, "sessions.json"),
	}, nil
}

func (s *Storage) Load() ([]*Instance, error) {
	instances, _, err := s.LoadAll()
	return instances, err
}

// LoadAll loads instances, groups, and settings
func (s *Storage) LoadAll() ([]*Instance, []*Group, error) {
	instances, groups, _, err := s.LoadAllWithSettings()
	return instances, groups, err
}

// LoadAllWithSettings loads instances, groups, and settings
func (s *Storage) LoadAllWithSettings() ([]*Instance, []*Group, *Settings, error) {
	data, err := os.ReadFile(s.configPath)
	if os.IsNotExist(err) {
		return []*Instance{}, []*Group{}, &Settings{}, nil
	}
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var storageData StorageData
	if err := json.Unmarshal(data, &storageData); err != nil {
		return nil, nil, nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Update status for all instances
	for _, instance := range storageData.Instances {
		instance.UpdateStatus()
	}

	if storageData.Groups == nil {
		storageData.Groups = []*Group{}
	}

	if storageData.Settings == nil {
		storageData.Settings = &Settings{}
	}

	return storageData.Instances, storageData.Groups, storageData.Settings, nil
}

func (s *Storage) Save(instances []*Instance) error {
	_, groups, settings, _ := s.LoadAllWithSettings()
	return s.SaveAll(instances, groups, settings)
}

// SaveWithGroups saves instances and groups (preserves settings)
func (s *Storage) SaveWithGroups(instances []*Instance, groups []*Group) error {
	_, _, settings, _ := s.LoadAllWithSettings()
	return s.SaveAll(instances, groups, settings)
}

// SaveSettings saves only the settings (preserves instances and groups)
func (s *Storage) SaveSettings(settings *Settings) error {
	instances, groups, _, _ := s.LoadAllWithSettings()
	return s.SaveAll(instances, groups, settings)
}

// SaveAll saves instances, groups, and settings
func (s *Storage) SaveAll(instances []*Instance, groups []*Group, settings *Settings) error {
	storageData := StorageData{
		Instances: instances,
		Groups:    groups,
		Settings:  settings,
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


// GetGroups returns all groups
func (s *Storage) GetGroups() ([]*Group, error) {
	_, groups, err := s.LoadAll()
	return groups, err
}

// AddGroup adds a new group
func (s *Storage) AddGroup(name string) (*Group, error) {
	instances, groups, err := s.LoadAll()
	if err != nil {
		return nil, err
	}

	// Check for duplicate names
	for _, g := range groups {
		if g.Name == name {
			return nil, fmt.Errorf("group with name '%s' already exists", name)
		}
	}

	group := &Group{
		ID:        fmt.Sprintf("grp_%d", time.Now().UnixNano()),
		Name:      name,
		Collapsed: false,
	}

	groups = append(groups, group)
	if err := s.SaveWithGroups(instances, groups); err != nil {
		return nil, err
	}

	return group, nil
}

// RemoveGroup removes a group (sessions become ungrouped)
func (s *Storage) RemoveGroup(id string) error {
	instances, groups, err := s.LoadAll()
	if err != nil {
		return err
	}

	// Ungroup all sessions in this group
	for _, inst := range instances {
		if inst.GroupID == id {
			inst.GroupID = ""
		}
	}

	// Remove the group
	newGroups := make([]*Group, 0, len(groups))
	found := false
	for _, g := range groups {
		if g.ID == id {
			found = true
			continue
		}
		newGroups = append(newGroups, g)
	}

	if !found {
		return fmt.Errorf("group not found")
	}

	return s.SaveWithGroups(instances, newGroups)
}

// RenameGroup renames a group
func (s *Storage) RenameGroup(id, name string) error {
	instances, groups, err := s.LoadAll()
	if err != nil {
		return err
	}

	for _, g := range groups {
		if g.ID == id {
			g.Name = name
			return s.SaveWithGroups(instances, groups)
		}
	}

	return fmt.Errorf("group not found")
}

// ToggleGroupCollapsed toggles the collapsed state of a group
func (s *Storage) ToggleGroupCollapsed(id string) error {
	instances, groups, err := s.LoadAll()
	if err != nil {
		return err
	}

	for _, g := range groups {
		if g.ID == id {
			g.Collapsed = !g.Collapsed
			return s.SaveWithGroups(instances, groups)
		}
	}

	return fmt.Errorf("group not found")
}

// SetInstanceGroup assigns an instance to a group
func (s *Storage) SetInstanceGroup(instanceID, groupID string) error {
	instances, groups, err := s.LoadAll()
	if err != nil {
		return err
	}

	for _, inst := range instances {
		if inst.ID == instanceID {
			inst.GroupID = groupID
			return s.SaveWithGroups(instances, groups)
		}
	}

	return fmt.Errorf("instance not found")
}
