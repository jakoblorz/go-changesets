package cli

import (
	"strconv"

	"github.com/jakoblorz/go-changesets/internal/workspace"
	"github.com/spf13/cobra"
)

const nodeDirtyFlag = "node-dangerously-activate-dirty-mode"

func workspaceOptionsFromCmd(cmd *cobra.Command) []workspace.Option {
	if nodeDirtyMode(cmd) {
		return []workspace.Option{workspace.WithNodeDirtyMode(true)}
	}

	return nil
}

func nodeDirtyMode(cmd *cobra.Command) bool {
	if cmd == nil {
		return false
	}

	flag := cmd.Flag(nodeDirtyFlag)
	if flag == nil {
		return false
	}

	enabled, err := strconv.ParseBool(flag.Value.String())
	if err != nil {
		return false
	}

	return enabled
}
