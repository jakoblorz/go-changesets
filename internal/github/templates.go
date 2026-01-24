package github

import (
	"bytes"
	"fmt"
	"path/filepath"
	"sync"
	"text/template"

	"github.com/Masterminds/sprig/v3"
	"github.com/jakoblorz/go-changesets/internal/filesystem"
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

const defaultBodyTemplate = `This PR was automatically generated for **{{.Project}}**.
{{- if gt (len .ChangelogPreview) 0}}

## 📋 Changes

{{.ChangelogPreview}}
{{- end}}
{{- if gt (len .RelatedPRs) 0}}

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

var (
	templateCache     = make(map[string]*template.Template)
	templateCacheLock sync.Mutex
)

func getBodyTemplate(fs filesystem.FileSystem, projectRoot string) (*template.Template, error) {
	path := findCustomBodyTemplate(fs, projectRoot)
	cacheKey := path
	if cacheKey == "" {
		cacheKey = "__default_github_pr_body__"
	}

	templateCacheLock.Lock()
	tmpl, ok := templateCache[cacheKey]
	templateCacheLock.Unlock()
	if ok {
		return tmpl, nil
	}

	var parsed *template.Template
	var err error
	if path != "" {
		data, readErr := fs.ReadFile(path)
		if readErr != nil {
			return nil, fmt.Errorf("failed to read github pr body template: %w", readErr)
		}
		parsed, err = template.New("github_pr_body").Funcs(sprig.TxtFuncMap()).Parse(string(data))
	} else {
		parsed, err = template.New("github_pr_body").Funcs(sprig.TxtFuncMap()).Parse(defaultBodyTemplate)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to parse github pr body template: %w", err)
	}

	templateCacheLock.Lock()
	templateCache[cacheKey] = parsed
	templateCacheLock.Unlock()

	return parsed, nil
}

func findCustomBodyTemplate(fs filesystem.FileSystem, start string) string {
	dir := start
	for {
		templatePath := filepath.Join(dir, ".changeset", "github_pr_body.tmpl")
		if fs.Exists(templatePath) {
			return templatePath
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}
		dir = parent
	}
}

const DefaultTitleTemplate = "🚀 Release {{.Project}} v{{.Version}}"

func getTitleTemplate(fs filesystem.FileSystem, projectRoot string) (*template.Template, error) {
	path := findCustomTitleTemplate(fs, projectRoot)
	cacheKey := path
	if cacheKey == "" {
		cacheKey = "__default_github_pr_title__"
	}

	templateCacheLock.Lock()
	tmpl, ok := templateCache[cacheKey]
	templateCacheLock.Unlock()
	if ok {
		return tmpl, nil
	}

	var parsed *template.Template
	var err error
	if path != "" {
		data, readErr := fs.ReadFile(path)
		if readErr != nil {
			return nil, fmt.Errorf("failed to read github pr title template: %w", readErr)
		}
		parsed, err = template.New("github_pr_title").Funcs(sprig.TxtFuncMap()).Parse(string(data))
	} else {
		parsed, err = template.New("github_pr_title").Funcs(sprig.TxtFuncMap()).Parse(DefaultTitleTemplate)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to parse github pr title template: %w", err)
	}

	templateCacheLock.Lock()
	templateCache[cacheKey] = parsed
	templateCacheLock.Unlock()

	return parsed, nil
}

func findCustomTitleTemplate(fs filesystem.FileSystem, start string) string {
	dir := start
	for {
		templatePath := filepath.Join(dir, ".changeset", "github_pr_title.tmpl")
		if fs.Exists(templatePath) {
			return templatePath
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}
		dir = parent
	}
}

type PRRenderer struct {
	fs filesystem.FileSystem
}

func NewPRRenderer(fs filesystem.FileSystem) *PRRenderer {
	return &PRRenderer{fs: fs}
}

func (p *PRRenderer) RenderTitle(data TemplateData, projectRoot string) (string, error) {
	root := projectRoot
	if root == "" {
		cwd, err := p.fs.Getwd()
		if err == nil {
			root = cwd
		}
	}

	tmpl, err := getTitleTemplate(p.fs, root)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute changelog template: %w", err)
	}

	return buf.String(), nil
}

func (p *PRRenderer) RenderBody(data TemplateData, projectRoot string) (string, error) {
	root := projectRoot
	if root == "" {
		cwd, err := p.fs.Getwd()
		if err == nil {
			root = cwd
		}
	}

	tmpl, err := getBodyTemplate(p.fs, root)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute changelog template: %w", err)
	}

	return buf.String(), nil
}
