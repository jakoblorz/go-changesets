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
							{ID: "change-a", Message: "analytics change from group A"},
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
							{ID: "change-b", Message: "analytics change from group B"},
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
}
