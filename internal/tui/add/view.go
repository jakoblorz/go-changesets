package add

import (
	"fmt"
	"strings"

	"github.com/jakoblorz/go-changesets/internal/tui"
)

// RenderSuccess renders a summary after a successful flow run.
func RenderSuccess(result *Result) string {
	var b strings.Builder

	b.WriteString(tui.SuccessStyle.Render("✓ Changeset Created"))
	b.WriteString("\n\n")
	b.WriteString(fmt.Sprintf("Created %d changeset file(s):\n", len(result.CreatedFiles)))
	for i, file := range result.CreatedFiles {
		b.WriteString(fmt.Sprintf("  %d. %s (%s: %s)\n", i+1, file, result.SelectedProjects[i], result.BumpType))
	}
	b.WriteString("\n")
	b.WriteString(fmt.Sprintf("Message: %s\n", result.Message))

	return b.String()
}
