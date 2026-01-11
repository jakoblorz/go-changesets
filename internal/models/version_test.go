package models

import (
	"testing"
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
			if err != nil {
				t.Fatalf("ParseVersion(%q) error: %v", tt.input, err)
			}

			if result.Major != tt.expected.Major ||
				result.Minor != tt.expected.Minor ||
				result.Patch != tt.expected.Patch ||
				result.Prerelease != tt.expected.Prerelease {
				t.Errorf("ParseVersion(%q) = %+v, want %+v", tt.input, result, tt.expected)
			}
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
			if err != nil {
				t.Fatalf("ParseVersion(%q) error: %v", tt.input, err)
			}

			if result.Major != tt.expected.Major ||
				result.Minor != tt.expected.Minor ||
				result.Patch != tt.expected.Patch ||
				result.Prerelease != tt.expected.Prerelease {
				t.Errorf("ParseVersion(%q) = %+v, want %+v", tt.input, result, tt.expected)
			}
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
			result := tt.version.String()
			if result != tt.expected {
				t.Errorf("Version.String() = %q, want %q", result, tt.expected)
			}
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
			result := tt.version.Tag()
			if result != tt.expected {
				t.Errorf("Version.Tag() = %q, want %q", result, tt.expected)
			}
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
			result := tt.v1.Compare(tt.v2)
			if result != tt.expected {
				t.Errorf("%s.Compare(%s) = %d, want %d",
					tt.v1.String(), tt.v2.String(), result, tt.expected)
			}
		})
	}
}

func TestVersion_WithPrerelease(t *testing.T) {
	original := &Version{1, 2, 3, ""}
	result := original.WithPrerelease("rc0")

	// Check new version has prerelease
	if result.Prerelease != "rc0" {
		t.Errorf("WithPrerelease() prerelease = %q, want %q", result.Prerelease, "rc0")
	}

	// Check original is unchanged
	if original.Prerelease != "" {
		t.Errorf("WithPrerelease() modified original version")
	}

	// Check other fields copied
	if result.Major != 1 || result.Minor != 2 || result.Patch != 3 {
		t.Errorf("WithPrerelease() did not copy version numbers correctly")
	}
}

func TestVersion_StripPrerelease(t *testing.T) {
	original := &Version{1, 2, 3, "rc0"}
	result := original.StripPrerelease()

	// Check new version has no prerelease
	if result.Prerelease != "" {
		t.Errorf("StripPrerelease() prerelease = %q, want empty", result.Prerelease)
	}

	// Check original is unchanged
	if original.Prerelease != "rc0" {
		t.Errorf("StripPrerelease() modified original version")
	}

	// Check other fields copied
	if result.Major != 1 || result.Minor != 2 || result.Patch != 3 {
		t.Errorf("StripPrerelease() did not copy version numbers correctly")
	}
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
			result := tt.version.IsPrerelease()
			if result != tt.expected {
				t.Errorf("IsPrerelease() = %v, want %v", result, tt.expected)
			}
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

			if result.Major != tt.expected.Major ||
				result.Minor != tt.expected.Minor ||
				result.Patch != tt.expected.Patch ||
				result.Prerelease != tt.expected.Prerelease {
				t.Errorf("Bump(%s) = %+v, want %+v", tt.bump, result, tt.expected)
			}

			// Original should be unchanged
			if original.Prerelease != "rc0" {
				t.Errorf("Bump() modified original version")
			}
		})
	}
}
