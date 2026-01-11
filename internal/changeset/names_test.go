package changeset

import (
	"strings"
	"testing"
)

func TestGenerateHumanFriendlyID(t *testing.T) {
	// Generate multiple IDs to test
	ids := make(map[string]bool)

	for i := 0; i < 10; i++ {
		id, err := generateHumanFriendlyID()
		if err != nil {
			t.Fatalf("failed to generate ID: %v", err)
		}

		// Check format: should have exactly 2 underscores
		parts := strings.Split(id, "_")
		if len(parts) != 3 {
			t.Errorf("expected 3 parts (adjective_animal_nanoid), got %d: %s", len(parts), id)
		}

		// Check first part is an adjective
		adjective := parts[0]
		if !contains(adjectives, adjective) {
			t.Errorf("first part should be an adjective, got: %s", adjective)
		}

		// Check second part is an animal
		animal := parts[1]
		if !contains(animals, animal) {
			t.Errorf("second part should be an animal, got: %s", animal)
		}

		// Check third part is nanoid (8 characters)
		nanoID := parts[2]
		if len(nanoID) != 8 {
			t.Errorf("nanoid should be 8 characters, got %d: %s", len(nanoID), nanoID)
		}

		// Check for uniqueness
		if ids[id] {
			t.Errorf("duplicate ID generated: %s", id)
		}
		ids[id] = true

		t.Logf("Generated ID: %s", id)
	}

	// All IDs should be unique
	if len(ids) != 10 {
		t.Errorf("expected 10 unique IDs, got %d", len(ids))
	}
}

func TestIDFormat(t *testing.T) {
	id, err := generateHumanFriendlyID()
	if err != nil {
		t.Fatalf("failed to generate ID: %v", err)
	}

	// Should match pattern: word_word_XXXXXXXX
	if len(id) < 10 { // minimum: a_b_12345678 = 12 chars
		t.Errorf("ID too short: %s", id)
	}

	// Should not start or end with underscore
	if strings.HasPrefix(id, "_") || strings.HasSuffix(id, "_") {
		t.Errorf("ID should not start or end with underscore: %s", id)
	}

	// Should not have consecutive underscores
	if strings.Contains(id, "__") {
		t.Errorf("ID should not have consecutive underscores: %s", id)
	}
}

// contains checks if a string slice contains a string
func contains(slice []string, str string) bool {
	for _, s := range slice {
		if s == str {
			return true
		}
	}
	return false
}
