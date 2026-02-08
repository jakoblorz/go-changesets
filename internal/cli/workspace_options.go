package cli

import (
	"strconv"

	"github.com/jakoblorz/go-changesets/internal/workspace"
	"github.com/spf13/cobra"
)

const nodeStrictWorkspaceFlag = "node-strict-workspace"

func workspaceOptionsFromCmd(cmd *cobra.Command) []workspace.Option {
	var opts []workspace.Option
	if nodeStrictWorkspace(cmd) {
		opts = append(opts, workspace.WithNodeStrictWorkspace(true))
	}

	if cmd != nil {
		opts = append(opts, workspace.WithWarningWriter(cmd.ErrOrStderr()))
	}

	return opts
}

func nodeStrictWorkspace(cmd *cobra.Command) bool {
	if cmd == nil {
		return false
	}

	flag := cmd.Flag(nodeStrictWorkspaceFlag)
	if flag == nil {
		return false
	}

	enabled, err := strconv.ParseBool(flag.Value.String())
	if err != nil {
		return false
	}

	return enabled
}
