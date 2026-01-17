package models

import (
	"testing"

	"github.com/stretchr/testify/require"
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

	require.True(t, cs.AffectsProject("project-a"))
	require.True(t, cs.AffectsProject("project-b"))
	require.False(t, cs.AffectsProject("project-c"))
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
	require.True(t, exists)
	require.Equal(t, BumpMajor, bump)

	bump, exists = cs.GetBumpForProject("project-b")
	require.True(t, exists)
	require.Equal(t, BumpMinor, bump)

	_, exists = cs.GetBumpForProject("nonexistent")
	require.False(t, exists)
}
