package add

import (
	"errors"
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	huh "github.com/charmbracelet/huh"
	"github.com/jakoblorz/go-changesets/internal/changeset"
	"github.com/jakoblorz/go-changesets/internal/filesystem"
	"github.com/jakoblorz/go-changesets/internal/models"
	"github.com/jakoblorz/go-changesets/internal/tui"
	"github.com/jakoblorz/go-changesets/internal/versioning"
	"github.com/jakoblorz/go-changesets/internal/workspace"
)

// Flow orchestrates the add command using huh forms.
type Flow struct {
	fs        filesystem.FileSystem
	workspace *workspace.Workspace
	csManager *changeset.Manager
	theme     *huh.Theme
}

// Result captures the successful output of the flow.
type Result struct {
	SelectedProjects []string
	BumpType         models.BumpType
	Message          string
	CreatedFiles     []string
}

// NewFlow constructs a Flow with the orange/blue huh theme.
func NewFlow(fs filesystem.FileSystem, ws *workspace.Workspace) *Flow {
	return &Flow{
		fs:        fs,
		workspace: ws,
		csManager: changeset.NewManager(fs, ws.ChangesetDir()),
		theme:     tui.NewHuhTheme(),
	}
}

// Run executes the forms sequentially; returns nil result on user abort.
func (f *Flow) Run() (*Result, error) {
	projects, err := f.selectProjects()
	if err != nil {
		if errors.Is(err, huh.ErrUserAborted) {
			return nil, nil
		}
		return nil, err
	}

	bumpType, err := f.selectBump(projects)
	if err != nil {
		if errors.Is(err, huh.ErrUserAborted) {
			return nil, nil
		}
		return nil, err
	}

	message, err := f.inputMessage()
	if err != nil {
		if errors.Is(err, huh.ErrUserAborted) {
			return nil, nil
		}
		return nil, err
	}

	createdFiles, err := f.createChangesets(projects, bumpType, message)
	if err != nil {
		return nil, err
	}

	return &Result{
		SelectedProjects: projects,
		BumpType:         bumpType,
		Message:          message,
		CreatedFiles:     createdFiles,
	}, nil
}

func (f *Flow) selectProjects() ([]string, error) {
	type projectOption struct {
		name     string
		kind     models.ProjectType
		version  string
		resolved bool
	}

	projects := f.workspace.Projects
	selected := make([]string, 0, len(projects))
	projectOptions := make([]projectOption, 0, len(projects))
	maxNameWidth := 0
	maxKindWidth := 0
	maxVersionWidth := 0

	for _, project := range projects {
		versionText, versionResolved := f.projectVersion(project)
		option := projectOption{
			name:     project.Name,
			kind:     project.Type,
			version:  versionText,
			resolved: versionResolved,
		}
		projectOptions = append(projectOptions, option)

		if width := len(option.name); width > maxNameWidth {
			maxNameWidth = width
		}
		if width := len(option.version); width > maxVersionWidth {
			maxVersionWidth = width
		}
		if width := len(string(option.kind)); width > maxKindWidth {
			maxKindWidth = width
		}
	}

	const columnGap = 2
	opts := make([]huh.Option[string], 0, len(projectOptions))
	for _, option := range projectOptions {
		namePadding := maxNameWidth - len(option.name)
		versionPadding := maxVersionWidth - len(option.version)
		kindLabel := string(option.kind)
		kindPadding := maxKindWidth - len(kindLabel)
		versionLabel := tui.SubtleStyle.Render(option.version)
		if !option.resolved {
			versionLabel = tui.SubtleErrorStyle.Render(option.version)
		}
		label := option.name +
			strings.Repeat(" ", namePadding+columnGap) +
			versionLabel +
			strings.Repeat(" ", versionPadding+columnGap) +
			renderProjectKind(option.kind) +
			strings.Repeat(" ", kindPadding)
		opts = append(opts, huh.NewOption(label, option.name))
	}

	keyMap := huh.NewDefaultKeyMap()
	keyMap.MultiSelect.Filter.SetEnabled(false)
	keyMap.MultiSelect.Toggle.SetKeys(" ")
	keyMap.MultiSelect.Toggle.SetHelp("space", "toggle selection")
	keyMap.MultiSelect.Submit.SetKeys("enter")
	keyMap.MultiSelect.Submit.SetHelp("enter", "continue")

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewMultiSelect[string]().
				Options(opts...).
				Value(&selected),
		).
			Title("Project Selection").
			Description("Select projects to include."),
	).
		WithTheme(f.theme).
		WithShowHelp(true).
		WithProgramOptions(tea.WithAltScreen()).
		WithKeyMap(keyMap)

	if err := form.Run(); err != nil {
		return nil, err
	}

	return selected, nil
}

func (f *Flow) selectBump(projects []string) (models.BumpType, error) {
	bump := ""

	opts := []huh.Option[string]{
		huh.NewOption("patch (0.0.X) — Bug fixes, no breaking changes", string(models.BumpPatch)),
		huh.NewOption("minor (0.X.0) — New features, backward compatible", string(models.BumpMinor)),
		huh.NewOption("major (X.0.0) — Breaking changes", string(models.BumpMajor)),
	}

	keyMap := huh.NewDefaultKeyMap()
	keyMap.Select.Filter.SetEnabled(false)
	keyMap.Select.Prev.SetEnabled(true)
	keyMap.Select.Submit.SetKeys("enter", " ")
	keyMap.Select.Submit.SetHelp("space/enter", "continue")

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Options(opts...).
				Value(&bump),
		).
			Title("Change Impact").
			Description(fmt.Sprintf("Applies to %s", strings.Join(projects, ", "))),
	).
		WithTheme(f.theme).
		WithShowHelp(true).
		WithProgramOptions(tea.WithAltScreen()).
		WithKeyMap(keyMap)

	if err := form.Run(); err != nil {
		return models.BumpType(""), err
	}

	parsed, err := models.ParseBumpType(bump)
	if err != nil {
		return models.BumpType(""), err
	}

	return parsed, nil
}

func (f *Flow) inputMessage() (string, error) {
	message := ""

	keyMap := huh.NewDefaultKeyMap()
	keyMap.Select.Submit.SetKeys("enter", " ")
	keyMap.Select.Submit.SetHelp("space/enter", "submit")

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewText().
				Lines(8).
				Value(&message).
				Placeholder("chore: update README.md to emphasize AI").
				Validate(func(v string) error {
					if strings.TrimSpace(v) == "" {
						return fmt.Errorf("message cannot be empty")
					}
					return nil
				}),
		).
			Title("Changelog Entry").
			Description("Describe your change. This will appear in the changelog later on."),
	).
		WithTheme(f.theme).
		WithShowHelp(true).
		WithProgramOptions(tea.WithAltScreen()).
		WithKeyMap(keyMap)

	if err := form.Run(); err != nil {
		return "", err
	}

	return strings.TrimSpace(message), nil
}

func (f *Flow) createChangesets(projects []string, bump models.BumpType, message string) ([]string, error) {
	created := make([]string, 0, len(projects))

	for _, projectName := range projects {
		id, err := f.csManager.GenerateID()
		if err != nil {
			return nil, fmt.Errorf("failed to generate changeset ID for %s: %w", projectName, err)
		}

		singleProjectBumps := map[string]models.BumpType{
			projectName: bump,
		}
		cs := models.NewChangeset(id, singleProjectBumps, message)

		if err := f.csManager.Write(cs); err != nil {
			return nil, fmt.Errorf("failed to write changeset for %s: %w", projectName, err)
		}

		created = append(created, fmt.Sprintf("%s.md", id))
	}

	return created, nil
}

func (f *Flow) projectVersion(project *models.Project) (string, bool) {
	versionStore := versioning.NewVersionStore(f.fs, project.Type)
	version, err := versionStore.Read(project.RootPath)
	if err != nil {
		return "version not resolved", false
	}
	return version.String(), true
}

func renderProjectKind(kind models.ProjectType) string {
	switch kind {
	case models.ProjectTypeGo:
		return tui.GoKindStyle.Render(string(kind))
	case models.ProjectTypeNode:
		return tui.NodeKindStyle.Render(string(kind))
	default:
		return string(kind)
	}
}
