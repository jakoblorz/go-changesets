package github

import (
	"encoding/json"
	"os"
	"time"
)

type PRMapping struct {
	Version   int                    `json:"version"`
	UpdatedAt string                 `json:"updatedAt"`
	Projects  map[string]PullRequest `json:"projects"`
}

func NewPRMapping() *PRMapping {
	return &PRMapping{
		Version:   1,
		UpdatedAt: time.Now().Format(time.RFC3339),
		Projects:  make(map[string]PullRequest),
	}
}

func ReadPRMapping(path string) (*PRMapping, error) {
	// TODO: make filesystem.FileSystem compatible
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

	if mapping.Projects == nil {
		mapping.Projects = make(map[string]PullRequest)
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

func (m *PRMapping) Set(project string, entry PullRequest) {
	m.Projects[project] = entry
}

func (m *PRMapping) Remove(project string) {
	delete(m.Projects, project)
}

func (m *PRMapping) Get(project string) (PullRequest, bool) {
	entry, ok := m.Projects[project]
	return entry, ok
}

func (m *PRMapping) Has(project string) bool {
	_, ok := m.Projects[project]
	return ok
}

func (m *PRMapping) IsEmpty() bool {
	return len(m.Projects) == 0
}
