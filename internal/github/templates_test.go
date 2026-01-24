package github

import (
	"testing"
	"text/template"

	"github.com/gkampitakis/go-snaps/snaps"
	"github.com/jakoblorz/go-changesets/internal/filesystem"
	"github.com/stretchr/testify/require"
)

func TestExecuteDefaultTemplate(t *testing.T) {
	t.Cleanup(resetChangelogTemplateCache)

	fs := filesystem.NewMockFileSystem()

	data := TemplateData{
		Project: "auth",
		Version: "1.2.0",
	}

	renderer := NewPRRenderer(fs)

	title, err := renderer.RenderTitle(data, "")
	require.NoError(t, err)
	snaps.MatchSnapshot(t, title)

	body, err := renderer.RenderBody(data, "")
	require.NoError(t, err)
	snaps.MatchSnapshot(t, body)
}

func TestExecuteDefaultTemplate_WithChangelogPreview(t *testing.T) {
	t.Cleanup(resetChangelogTemplateCache)

	fs := filesystem.NewMockFileSystem()

	data := TemplateData{
		Project:          "auth",
		Version:          "1.2.0",
		ChangelogPreview: "### Minor Changes\n- Add OAuth2 support",
	}

	renderer := NewPRRenderer(fs)
	body, err := renderer.RenderBody(data, "")
	require.NoError(t, err)
	snaps.MatchSnapshot(t, body)
}

func TestExecuteDefaultTemplate_WithRelatedPRs(t *testing.T) {
	t.Cleanup(resetChangelogTemplateCache)

	fs := filesystem.NewMockFileSystem()

	data := TemplateData{
		Project:          "auth",
		Version:          "1.2.0",
		ChangelogPreview: "### Minor Changes\n- Add OAuth2 support",
		RelatedPRs: []RelatedPRInfo{
			{Number: 123, Project: "auth", Version: "1.2.0"},
			{Number: 124, Project: "api", Version: "2.1.0"},
		},
	}

	renderer := NewPRRenderer(fs)
	body, err := renderer.RenderBody(data, "")
	require.NoError(t, err)
	snaps.MatchSnapshot(t, body)
}

func TestExecuteDefaultTemplate_WithCustomTemplates(t *testing.T) {
	t.Cleanup(resetChangelogTemplateCache)

	fs := filesystem.NewMockFileSystem()
	fs.AddFile("/workspace/.changeset/github_pr_title.tmpl", []byte(`Let's go, let's release {{.Project}} at v{{.Version}}`))
	fs.AddFile("/workspace/.changeset/github_pr_body.tmpl", []byte(`Release already! {{.Project}} v{{.Version}}`))

	data := TemplateData{
		Project:          "auth",
		Version:          "1.2.0",
		ChangelogPreview: "### Minor Changes\n- Add OAuth2 support",
		RelatedPRs: []RelatedPRInfo{
			{Number: 123, Project: "auth", Version: "1.2.0"},
			{Number: 124, Project: "api", Version: "2.1.0"},
		},
	}

	renderer := NewPRRenderer(fs)

	title, err := renderer.RenderTitle(data, "")
	require.NoError(t, err)
	snaps.MatchSnapshot(t, title)

	body, err := renderer.RenderBody(data, "")
	require.NoError(t, err)
	snaps.MatchSnapshot(t, body)
}

func resetChangelogTemplateCache() {
	templateCacheLock.Lock()
	defer templateCacheLock.Unlock()
	templateCache = make(map[string]*template.Template)
}
