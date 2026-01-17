package models

import (
	"testing"
)

func TestChangeset_AffectsProject(t *testing.T) {
	cs := &Changeset{
		ID:      "test",
		Message: "Test",
		Projects: map[string]BumpType{
			"project-a": BumpMajor,
			"project-b": BumpPatch,
		},
	}

	if !cs.AffectsProject("project-a") {
		t.Error("AffectsProject(project-a) = false, want true")
	}

	if !cs.AffectsProject("project-b") {
		t.Error("AffectsProject(project-b) = false, want true")
	}

	if cs.AffectsProject("project-c") {
		t.Error("AffectsProject(project-c) = true, want false")
	}
}

func TestChangeset_GetBumpForProject(t *testing.T) {
	cs := &Changeset{
		ID:      "test",
		Message: "Test",
		Projects: map[string]BumpType{
			"project-a": BumpMajor,
			"project-b": BumpMinor,
		},
	}

	bump, exists := cs.GetBumpForProject("project-a")
	if !exists {
		t.Error("GetBumpForProject(project-a) exists = false, want true")
	}
	if bump != BumpMajor {
		t.Errorf("GetBumpForProject(project-a) = %s, want %s", bump, BumpMajor)
	}

	bump, exists = cs.GetBumpForProject("project-b")
	if !exists {
		t.Error("GetBumpForProject(project-b) exists = false, want true")
	}
	if bump != BumpMinor {
		t.Errorf("GetBumpForProject(project-b) = %s, want %s", bump, BumpMinor)
	}

	_, exists = cs.GetBumpForProject("nonexistent")
	if exists {
		t.Error("GetBumpForProject(nonexistent) exists = true, want false")
	}
}
