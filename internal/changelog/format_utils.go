package changelog

import (
	"strings"

	"github.com/jakoblorz/go-changesets/internal/models"

	"github.com/jakoblorz/go-changesets/internal/changeset"
)

type formattedSection struct {
	Title      string
	Changesets []*models.Changeset
}

func buildSections(changesets []*models.Changeset, projectName string) []formattedSection {
	relevant := changesets
	if projectName != "" {
		relevant = changeset.FilterByProject(changesets, projectName)
	}

	var major, minor, patch []*models.Changeset
	for _, cs := range relevant {
		bump, ok := determineBump(cs, projectName)
		if !ok {
			continue
		}

		switch bump {
		case models.BumpMajor:
			major = append(major, cs)
		case models.BumpMinor:
			minor = append(minor, cs)
		case models.BumpPatch:
			patch = append(patch, cs)
		}
	}

	sections := make([]formattedSection, 0, 3)
	if len(major) > 0 {
		sections = append(sections, formattedSection{Title: "Major Changes", Changesets: major})
	}
	if len(minor) > 0 {
		sections = append(sections, formattedSection{Title: "Minor Changes", Changesets: minor})
	}
	if len(patch) > 0 {
		sections = append(sections, formattedSection{Title: "Patch Changes", Changesets: patch})
	}

	return sections
}

func determineBump(cs *models.Changeset, projectName string) (models.BumpType, bool) {
	if projectName != "" {
		bump, ok := cs.GetBumpForProject(projectName)
		return bump, ok
	}

	for _, bump := range cs.Projects {
		return bump, true
	}

	return "", false
}

func splitMessage(message string) (string, []string) {
	message = strings.TrimSpace(message)
	if message == "" {
		return "", nil
	}

	lines := strings.Split(message, "\n")
	if len(lines) == 0 {
		return "", nil
	}

	firstLine := lines[0]
	if firstLine == "" {
		return "", nil
	}

	var rest []string
	if len(lines) > 1 {
		for i := 1; i < len(lines); i++ {
			if strings.TrimSpace(lines[i]) == "" {
				continue
			}
			rest = append(rest, strings.TrimSpace(lines[i]))
		}
	}

	return firstLine, rest
}
