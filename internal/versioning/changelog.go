package versioning

import (
	"bytes"
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"text/template"
	"time"

	"github.com/jakoblorz/go-changesets/internal/filesystem"
	"github.com/jakoblorz/go-changesets/internal/models"
)

const changelogFileName = "CHANGELOG.md"

const defaultChangelogTemplate = `{{- if .Version}}## {{.Version}} ({{.Date}})
{{end}}
{{range $index, $section := .Sections}}### {{$section.Title}}
{{range $section.Items}}
- {{.FirstLine}}{{if .PR}} ([#{{.PR.Number}}]({{.PR.URL}}) by @{{.PR.Author}}){{end}}{{if .RestLines}}
{{range .RestLines}}  {{.}}
{{end}}{{end}}{{end}}

{{end}}`

var (
	templateCache     = make(map[string]*template.Template)
	templateCacheLock sync.Mutex
)

func getChangelogTemplate(fs filesystem.FileSystem, projectRoot string) (*template.Template, error) {
	path := findCustomTemplate(fs, projectRoot)
	cacheKey := path
	if cacheKey == "" {
		cacheKey = "__default__"
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
			return nil, fmt.Errorf("failed to read changelog template: %w", readErr)
		}
		parsed, err = template.New("changelog").Parse(string(data))
	} else {
		parsed, err = template.New("changelog").Parse(defaultChangelogTemplate)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to parse changelog template: %w", err)
	}

	templateCacheLock.Lock()
	templateCache[cacheKey] = parsed
	templateCacheLock.Unlock()

	return parsed, nil
}

func findCustomTemplate(fs filesystem.FileSystem, start string) string {
	dir := start
	for {
		templatePath := filepath.Join(dir, ".changeset", "changelog.tmpl")
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

// Changelog handles reading and writing CHANGELOG.md files
type Changelog struct {
	fs filesystem.FileSystem
}

// NewChangelog creates a new Changelog instance
func NewChangelog(fs filesystem.FileSystem) *Changelog {
	return &Changelog{fs: fs}
}

// ChangelogEntry represents an entry to be added to the changelog
type ChangelogEntry struct {
	Version    *models.Version
	Date       time.Time
	Changesets []*models.Changeset
}

// Append adds a new entry to the changelog
func (cl *Changelog) Append(projectRoot string, entry *ChangelogEntry) error {
	changelogPath := filepath.Join(projectRoot, changelogFileName)

	var existingContent string
	if cl.fs.Exists(changelogPath) {
		data, err := cl.fs.ReadFile(changelogPath)
		if err != nil {
			return fmt.Errorf("failed to read changelog: %w", err)
		}
		existingContent = string(data)
	}

	newEntry, err := cl.formatEntry(entry, projectRoot)
	if err != nil {
		return err
	}

	var buf bytes.Buffer

	if strings.Contains(existingContent, "# Changelog") {
		lines := strings.Split(existingContent, "\n")
		headerEndIdx := 0
		for i, line := range lines {
			if strings.HasPrefix(line, "## ") {
				headerEndIdx = i
				break
			}
		}
		if headerEndIdx > 0 {
			buf.WriteString(strings.Join(lines[0:headerEndIdx], "\n"))
			buf.WriteString("\n")
		}
	} else {
		buf.WriteString("# Changelog\n\n")
		buf.WriteString("All notable changes to this project will be documented in this file.\n\n")
	}

	buf.WriteString(newEntry)
	buf.WriteString("\n")

	if strings.Contains(existingContent, "# Changelog") {
		lines := strings.Split(existingContent, "\n")
		for i, line := range lines {
			if strings.HasPrefix(line, "## ") {
				buf.WriteString(strings.Join(lines[i:], "\n"))
				break
			}
		}
	} else {
		buf.WriteString(existingContent)
	}

	if err := cl.fs.WriteFile(changelogPath, buf.Bytes(), 0644); err != nil {
		return fmt.Errorf("failed to write changelog: %w", err)
	}

	return nil
}

// FormatEntry generates changelog content without version header.
func (cl *Changelog) FormatEntry(changesets []*models.Changeset, projectName, projectRoot string) (string, error) {
	return cl.formatWithTemplate(changesets, projectName, projectRoot, "", time.Time{})
}

func (cl *Changelog) formatEntry(entry *ChangelogEntry, projectRoot string) (string, error) {
	return cl.formatWithTemplate(entry.Changesets, "", projectRoot, entry.Version.String(), entry.Date)
}

func (cl *Changelog) formatWithTemplate(changesets []*models.Changeset, projectName, projectRoot, version string, date time.Time) (string, error) {
	root := projectRoot
	if root == "" {
		cwd, err := cl.fs.Getwd()
		if err == nil {
			root = cwd
		}
	}

	tmpl, err := getChangelogTemplate(cl.fs, root)
	if err != nil {
		return "", err
	}

	data := cl.buildTemplateData(changesets, projectName, version, date)

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute changelog template: %w", err)
	}

	return strings.TrimSpace(buf.String()), nil
}

type changelogTemplateData struct {
	Project  string
	Version  string
	Date     string
	Sections []changelogTemplateSection
}

type changelogTemplateSection struct {
	Title string
	Items []changelogTemplateItem
}

type changelogTemplateItem struct {
	FirstLine string
	RestLines []string
	PR        *models.PullRequest
}

func (cl *Changelog) buildTemplateData(changesets []*models.Changeset, projectName, version string, date time.Time) changelogTemplateData {
	data := changelogTemplateData{
		Project: projectName,
		Version: version,
	}
	if !date.IsZero() {
		data.Date = date.Format("2006-01-02")
	}

	sections := buildSections(changesets, projectName)
	for _, section := range sections {
		data.Sections = append(data.Sections, changelogTemplateSection{
			Title: section.Title,
			Items: buildTemplateItems(section.Changesets),
		})
	}

	return data
}

func buildTemplateItems(changesets []*models.Changeset) []changelogTemplateItem {
	items := make([]changelogTemplateItem, 0, len(changesets))

	for _, cs := range changesets {
		first, rest := splitMessage(cs.Message)
		if first == "" {
			continue
		}

		items = append(items, changelogTemplateItem{
			FirstLine: first,
			RestLines: rest,
			PR:        cs.PR,
		})
	}

	return items
}

// GetEntryForVersion reads the changelog entry for a specific version
func (cl *Changelog) GetEntryForVersion(projectRoot string, version *models.Version) (string, error) {
	changelogPath := filepath.Join(projectRoot, changelogFileName)

	if !cl.fs.Exists(changelogPath) {
		return "", fmt.Errorf("changelog not found")
	}

	data, err := cl.fs.ReadFile(changelogPath)
	if err != nil {
		return "", fmt.Errorf("failed to read changelog: %w", err)
	}

	content := string(data)
	versionHeader := fmt.Sprintf("## %s", version.String())

	startIdx := strings.Index(content, versionHeader)
	if startIdx == -1 {
		return "", fmt.Errorf("version %s not found in changelog", version.String())
	}

	endIdx := strings.Index(content[startIdx+len(versionHeader):], "\n## ")
	if endIdx == -1 {
		return strings.TrimSpace(content[startIdx:]), nil
	}

	endIdx += startIdx + len(versionHeader)
	return strings.TrimSpace(content[startIdx:endIdx]), nil
}
