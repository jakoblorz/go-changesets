package changeset

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGenerateHumanFriendlyID(t *testing.T) {
	ids := make(map[string]bool)

	for i := 0; i < 10; i++ {
		id, err := generateHumanFriendlyID()
		require.NoError(t, err)

		parts := strings.Split(id, "_")
		require.Len(t, parts, 3, "expected adjective_animal_nanoid format: %s", id)

		require.Truef(t, contains(adjectives, parts[0]), "first part should be adjective: %s", parts[0])
		require.Truef(t, contains(animals, parts[1]), "second part should be animal: %s", parts[1])
		require.Len(t, parts[2], 8, "nanoid portion wrong length: %s", parts[2])

		require.Falsef(t, ids[id], "duplicate ID generated: %s", id)
		ids[id] = true
	}

	require.Len(t, ids, 10)
}

func TestIDFormat(t *testing.T) {
	id, err := generateHumanFriendlyID()
	require.NoError(t, err)

	require.GreaterOrEqual(t, len(id), 12, "ID too short: %s", id)
	require.False(t, strings.HasPrefix(id, "_"), "ID should not start with underscore: %s", id)
	require.False(t, strings.HasSuffix(id, "_"), "ID should not end with underscore: %s", id)
	require.False(t, strings.Contains(id, "__"), "ID should not have consecutive underscores: %s", id)
}

func contains(slice []string, str string) bool {
	for _, s := range slice {
		if s == str {
			return true
		}
	}
	return false
}
