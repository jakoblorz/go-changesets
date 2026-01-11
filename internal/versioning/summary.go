package versioning

import (
	"bytes"
	"fmt"

	"github.com/jakoblorz/go-changesets/internal/models"
)

// GenerateChangesetSummary generates a markdown summary of changesets.
// This is used for GitHub release notes (especially for snapshot releases).
func GenerateChangesetSummary(changesets []*models.Changeset, projectName string) string {
	var buf bytes.Buffer

	sections := buildSections(changesets, projectName)
	for _, section := range sections {
		buf.WriteString(fmt.Sprintf("### %s\n\n", section.Title))
		for _, cs := range section.Changesets {
			writeChangesetEntry(&buf, cs)
		}
		buf.WriteString("\n")
	}

	buf.WriteString("---\n")
	buf.WriteString("**This is a pre-release snapshot for testing purposes.**\n")

	return buf.String()
}

func writeChangesetEntry(buf *bytes.Buffer, cs *models.Changeset) {
	first, rest := splitMessage(cs.Message)
	if first == "" {
		return
	}

	prSuffix := cs.FormatPRSuffix(true)
	if prSuffix != "" {
		buf.WriteString(fmt.Sprintf("- %s %s\n", first, prSuffix))
	} else {
		buf.WriteString(fmt.Sprintf("- %s\n", first))
	}

	for _, line := range rest {
		buf.WriteString(fmt.Sprintf("  %s\n", line))
	}
}
