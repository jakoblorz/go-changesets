package github

import (
	"encoding/json"
	"os"
	"time"
)

type PRMapping struct {
	Version   int                `json:"version"`
	UpdatedAt string             `json:"updatedAt"`
	Mappings  map[string]PREntry `json:"mappings"`
}

type PREntry struct {
	PRNumber         int    `json:"prNumber"`
	Branch           string `json:"branch"`
	Version          string `json:"version"`
	ChangelogPreview string `json:"changelogPreview"`
}

func NewPRMapping() *PRMapping {
	return &PRMapping{
		Version:   1,
		UpdatedAt: time.Now().Format(time.RFC3339),
		Mappings:  make(map[string]PREntry),
	}
}

func ReadPRMapping(path string) (*PRMapping, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return NewPRMapping(), nil
		}
		return nil, err
	}

	if len(data) == 0 {
		return NewPRMapping(), nil
	}

	var mapping PRMapping
	if err := json.Unmarshal(data, &mapping); err != nil {
		return nil, err
	}

	if mapping.Mappings == nil {
		mapping.Mappings = make(map[string]PREntry)
	}

	return &mapping, nil
}

func (m *PRMapping) Write(path string) error {
	m.UpdatedAt = time.Now().Format(time.RFC3339)
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

func (m *PRMapping) Set(project string, entry PREntry) {
	m.Mappings[project] = entry
}

func (m *PRMapping) Remove(project string) {
	delete(m.Mappings, project)
}

func (m *PRMapping) Get(project string) (PREntry, bool) {
	entry, ok := m.Mappings[project]
	return entry, ok
}

func (m *PRMapping) Has(project string) bool {
	_, ok := m.Mappings[project]
	return ok
}

func (m *PRMapping) IsEmpty() bool {
	return len(m.Mappings) == 0
}
