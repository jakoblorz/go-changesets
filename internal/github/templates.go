package github

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"text/template"
)

type TemplateData struct {
	Project          string
	Version          string
	CurrentVersion   string
	ChangelogPreview string
	CommitSHA        string
	RelatedPRs       []RelatedPRInfo
}

type RelatedPRInfo struct {
	Number  int
	Project string
	Version string
}

const DefaultBodyTemplate = "This PR was automatically generated for **{{.Project}}**.\n\n## 📋 Changes\n\n{{.ChangelogPreview}}\n\n<!-- RELATED_PRS_PLACEHOLDER -->\n\n## 📦 What happens when you merge?\n- Version bumped to **{{.Version}}**\n- Changelog updated\n- Changesets consumed\n- Publish workflow creates GitHub release: `{{.Project}}@{{.Version}}`"

const DefaultTitleTemplate = "🚀 Release {{.Project}} v{{.Version}}"

func ParseTemplateFile(path string) (*template.Template, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	return template.New(filepath.Base(path)).Parse(string(data))
}

func ExecuteTemplate(tmpl *template.Template, name string, data interface{}) (string, error) {
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func ExecuteDefaultTemplate(name string, data interface{}) (string, error) {
	var tmplStr string
	switch name {
	case "body":
		tmplStr = DefaultBodyTemplate
	case "title":
		tmplStr = DefaultTitleTemplate
	default:
		return "", fmt.Errorf("unknown template: %s", name)
	}

	tmpl, err := template.New(name).Parse(tmplStr)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func BuildRelatedPRsSection(commitSHA string, relatedPRs []RelatedPRInfo, currentProject string) string {
	if len(relatedPRs) <= 1 {
		return ""
	}

	var result string
	result += "## 🔗 Related Release PRs\n\n"
	result += fmt.Sprintf("This release is part of a coordinated change from commit [`%s`]:\n", commitSHA)
	for _, pr := range relatedPRs {
		marker := ""
		if pr.Project == currentProject {
			marker = "← **You are here**"
		}
		result += fmt.Sprintf("- #%d Release %s v%s %s\n", pr.Number, pr.Project, pr.Version, marker)
	}
	result += "\n**Tip:** These changes were made together. Consider reviewing and merging all related PRs together.\n"

	return result
}

func ReplaceRelatedPRsPlaceholder(body, replacement string) string {
	startMarker := "<!-- RELATED_PRS_PLACEHOLDER -->"

	// If no placeholder found, append at the end
	if !containsString(body, startMarker) {
		return body + "\n\n" + replacement
	}

	// Replace the placeholder with the replacement content
	return replaceString(body, startMarker, replacement)
}

func replaceString(s, old, new string) string {
	result := ""
	idx := -1
	for i := 0; i <= len(s)-len(old); i++ {
		if s[i:i+len(old)] == old {
			result += s[idx+1 : i]
			result += new
			idx = i + len(old) - 1
		}
	}
	if idx < len(s)-1 {
		result += s[idx+1:]
	}
	return result
}

func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsSubstring(s, substr))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
