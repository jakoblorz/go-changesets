package cli

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTreeOutput_GetGroupForProject_AggregatesAcrossGroups(t *testing.T) {
	tree := TreeOutput{
		Groups: []ChangesetGroup{
			{
				Commit: "commit-a",
				Projects: []ProjectChangesetsInfo{
					{
						Name: "analytics",
						Changesets: []ChangesetInfo{
							{ID: "change-a", Message: "analytics change from group A", PR: &PullRequestInfo{Number: 10, Labels: []string{"release", "backend"}}},
						},
					},
					{
						Name: "www",
						Changesets: []ChangesetInfo{
							{ID: "change-a", Message: "www change from group A"},
						},
					},
				},
			},
			{
				Commit: "commit-b",
				Projects: []ProjectChangesetsInfo{
					{
						Name: "analytics",
						Changesets: []ChangesetInfo{
							{ID: "change-b", Message: "analytics change from group B", PR: &PullRequestInfo{Number: 11, Labels: []string{"release", "analytics"}}},
						},
					},
					{
						Name: "bookkeeper",
						Changesets: []ChangesetInfo{
							{ID: "change-b", Message: "bookkeeper change from group B"},
						},
					},
				},
			},
		},
	}

	group := tree.GetGroupForProject("analytics")
	require.NotNil(t, group)
	require.Len(t, group.Projects, 3)

	projects := map[string]int{}
	for _, project := range group.Projects {
		projects[project.Name] = len(project.Changesets)
	}

	require.Equal(t, 2, projects["analytics"])
	require.Equal(t, 1, projects["www"])
	require.Equal(t, 1, projects["bookkeeper"])

	analyticsLabels := map[string][]string{}
	for _, project := range group.Projects {
		if project.Name != "analytics" {
			continue
		}
		for _, cs := range project.Changesets {
			if cs.PR != nil {
				analyticsLabels[cs.ID] = cs.PR.Labels
			}
		}
	}

	require.Equal(t, []string{"release", "backend"}, analyticsLabels["change-a"])
	require.Equal(t, []string{"release", "analytics"}, analyticsLabels["change-b"])
}
