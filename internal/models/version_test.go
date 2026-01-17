package models

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseVersion_WithPrerelease(t *testing.T) {
	tests := []struct {
		input    string
		expected *Version
	}{
		{"1.2.3-rc0", &Version{1, 2, 3, "rc0"}},
		{"v1.2.3-rc0", &Version{1, 2, 3, "rc0"}},
		{"0.1.0-rc5", &Version{0, 1, 0, "rc5"}},
		{"2.0.0-beta.1", &Version{2, 0, 0, "beta.1"}},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result, err := ParseVersion(tt.input)
			require.NoError(t, err)
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestParseVersion_WithoutPrerelease(t *testing.T) {
	tests := []struct {
		input    string
		expected *Version
	}{
		{"1.2.3", &Version{1, 2, 3, ""}},
		{"v1.2.3", &Version{1, 2, 3, ""}},
		{"0.0.0", &Version{0, 0, 0, ""}},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result, err := ParseVersion(tt.input)
			require.NoError(t, err)
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestVersion_String_WithPrerelease(t *testing.T) {
	tests := []struct {
		version  *Version
		expected string
	}{
		{&Version{1, 2, 3, "rc0"}, "1.2.3-rc0"},
		{&Version{0, 1, 0, "rc5"}, "0.1.0-rc5"},
		{&Version{2, 0, 0, "beta.1"}, "2.0.0-beta.1"},
		{&Version{1, 2, 3, ""}, "1.2.3"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			require.Equal(t, tt.expected, tt.version.String())
		})
	}
}

func TestVersion_Tag_WithPrerelease(t *testing.T) {
	tests := []struct {
		version  *Version
		expected string
	}{
		{&Version{1, 2, 3, "rc0"}, "v1.2.3-rc0"},
		{&Version{0, 1, 0, "rc5"}, "v0.1.0-rc5"},
		{&Version{2, 0, 0, "beta.1"}, "v2.0.0-beta.1"},
		{&Version{1, 2, 3, ""}, "v1.2.3"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			require.Equal(t, tt.expected, tt.version.Tag())
		})
	}
}

func TestVersion_Compare_PrereleaseOrdering(t *testing.T) {
	tests := []struct {
		name     string
		v1       *Version
		v2       *Version
		expected int
	}{
		// Prerelease < release
		{"rc0 < release", &Version{1, 2, 3, "rc0"}, &Version{1, 2, 3, ""}, -1},
		{"release > rc0", &Version{1, 2, 3, ""}, &Version{1, 2, 3, "rc0"}, 1},

		// RC number ordering
		{"rc0 < rc1", &Version{1, 2, 3, "rc0"}, &Version{1, 2, 3, "rc1"}, -1},
		{"rc1 > rc0", &Version{1, 2, 3, "rc1"}, &Version{1, 2, 3, "rc0"}, 1},
		{"rc0 == rc0", &Version{1, 2, 3, "rc0"}, &Version{1, 2, 3, "rc0"}, 0},

		// Base version comparison takes precedence
		{"1.2.3-rc0 < 1.2.4", &Version{1, 2, 3, "rc0"}, &Version{1, 2, 4, ""}, -1},
		{"1.2.4 > 1.2.3-rc0", &Version{1, 2, 4, ""}, &Version{1, 2, 3, "rc0"}, 1},
		{"1.2.3 < 1.2.4-rc0", &Version{1, 2, 3, ""}, &Version{1, 2, 4, "rc0"}, -1},

		// Same base version, same prerelease
		{"same version", &Version{1, 2, 3, ""}, &Version{1, 2, 3, ""}, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.expected, tt.v1.Compare(tt.v2), "%s vs %s", tt.v1.String(), tt.v2.String())
		})
	}
}

func TestVersion_WithPrerelease(t *testing.T) {
	original := &Version{1, 2, 3, ""}
	result := original.WithPrerelease("rc0")

	require.Equal(t, "rc0", result.Prerelease)
	require.Equal(t, "", original.Prerelease)
	require.Equal(t, &Version{1, 2, 3, "rc0"}, result)
}

func TestVersion_StripPrerelease(t *testing.T) {
	original := &Version{1, 2, 3, "rc0"}
	result := original.StripPrerelease()

	// Check new version has no prerelease
	require.Equal(t, "", result.Prerelease)
	require.Equal(t, "rc0", original.Prerelease)
	require.Equal(t, &Version{1, 2, 3, ""}, result)
}

func TestVersion_IsPrerelease(t *testing.T) {
	tests := []struct {
		name     string
		version  *Version
		expected bool
	}{
		{"with rc", &Version{1, 2, 3, "rc0"}, true},
		{"with beta", &Version{1, 2, 3, "beta.1"}, true},
		{"without prerelease", &Version{1, 2, 3, ""}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.expected, tt.version.IsPrerelease())
		})
	}
}

func TestVersion_Bump_WithPrerelease(t *testing.T) {
	// Bump should strip prerelease and apply the bump
	original := &Version{1, 2, 3, "rc0"}

	tests := []struct {
		bump     BumpType
		expected *Version
	}{
		{BumpPatch, &Version{1, 2, 4, ""}},
		{BumpMinor, &Version{1, 3, 0, ""}},
		{BumpMajor, &Version{2, 0, 0, ""}},
	}

	for _, tt := range tests {
		t.Run(string(tt.bump), func(t *testing.T) {
			result := original.Bump(tt.bump)
			require.Equal(t, tt.expected, result)
			require.Equal(t, "rc0", original.Prerelease)
		})
	}
}
