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
	RelatedPRs       []RelatedPRInfo
}

type RelatedPRInfo struct {
	Number  int
	Project string
	Version string
}

const DefaultBodyTemplate = `This PR was automatically generated for **{{.Project}}**.

## 📋 Changes

{{.ChangelogPreview}}
{{- if .RelatedPRs}}

## 🔗 Related Release PRs

This release is part of a coordinated change:
{{range .RelatedPRs}}- #{{.Number}} Release {{.Project}} v{{.Version}}
{{end}}
{{- end}}

## 📦 What happens when you merge?
- Version bumped to **{{.Version}}**
- Changelog updated
- Changesets consumed
- Publish workflow creates GitHub release: ` + "`" + `{{.Project}}@{{.Version}}` + "`" + `
`

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
