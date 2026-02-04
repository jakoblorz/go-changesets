package cli

import (
	"fmt"

	"github.com/jakoblorz/go-changesets/internal/models"
)

func tagPrefixPattern(projectName string, projectType models.ProjectType) string {
	if projectType == models.ProjectTypeNode {
		return fmt.Sprintf("%s@*", projectName)
	}
	return fmt.Sprintf("%s@v*", projectName)
}

func tagVersionString(projectType models.ProjectType, version *models.Version) string {
	if projectType == models.ProjectTypeNode {
		return version.String()
	}
	return version.Tag()
}

func tagName(projectName string, projectType models.ProjectType, version *models.Version) string {
	return fmt.Sprintf("%s@%s", projectName, tagVersionString(projectType, version))
}
