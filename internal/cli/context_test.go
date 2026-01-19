package cli

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/jakoblorz/go-changesets/internal/models"
	"github.com/stretchr/testify/require"
)

func TestReadProjectContextFromStdin(t *testing.T) {
	t.Run("valid JSON context", func(t *testing.T) {
		ctx := &models.ProjectContext{
			Project:        "my-project",
			ProjectPath:    "/workspace/my-project",
			ModulePath:     "github.com/example/my-project",
			CurrentVersion: "1.0.0",
			LatestTag:      "0.9.0",
			HasChangesets:  true,
			IsOutdated:     true,
		}
		data, err := json.Marshal(ctx)
		require.NoError(t, err)

		err = runWithStdin(string(data), func() error {
			actual, err := readProjectContextFromStdin()
			if err != nil {
				return err
			}
			require.Equal(t, ctx.Project, actual.Project)
			require.Equal(t, ctx.ProjectPath, actual.ProjectPath)
			require.Equal(t, ctx.ModulePath, actual.ModulePath)
			return nil
		})
		require.NoError(t, err)
	})

	t.Run("missing project name", func(t *testing.T) {
		ctx := &models.ProjectContext{
			ProjectPath:    "/workspace/my-project",
			ModulePath:     "github.com/example/my-project",
			CurrentVersion: "1.0.0",
		}
		data, err := json.Marshal(ctx)
		require.NoError(t, err)

		err = runWithStdin(string(data), func() error {
			_, err := readProjectContextFromStdin()
			return err
		})
		require.Error(t, err)
		require.Contains(t, err.Error(), "project name is required")
	})

	t.Run("invalid JSON", func(t *testing.T) {
		err := runWithStdin("not valid json", func() error {
			_, err := readProjectContextFromStdin()
			return err
		})
		require.Error(t, err)
		require.Contains(t, err.Error(), "failed to parse context JSON")
	})

	t.Run("empty stdin", func(t *testing.T) {
		err := runWithStdin("", func() error {
			_, err := readProjectContextFromStdin()
			return err
		})
		require.Error(t, err)
		require.Contains(t, err.Error(), "empty STDIN")
	})

	t.Run("terminal detection", func(t *testing.T) {
		err := runWithStdinViaTerminal(func() error {
			_, err := readProjectContextFromStdin()
			return err
		})
		require.Error(t, err)
		require.Contains(t, err.Error(), "no context on STDIN")
	})
}

func TestReadProjectContextFromEnv(t *testing.T) {
	t.Run("valid JSON context", func(t *testing.T) {
		ctx := &models.ProjectContext{
			Project:        "my-project",
			ProjectPath:    "/workspace/my-project",
			ModulePath:     "github.com/example/my-project",
			CurrentVersion: "1.0.0",
			LatestTag:      "0.9.0",
			HasChangesets:  true,
			IsOutdated:     true,
		}
		data, err := json.Marshal(ctx)
		require.NoError(t, err)

		orig := os.Getenv("CHANGESET_CONTEXT")
		defer os.Setenv("CHANGESET_CONTEXT", orig)

		os.Setenv("CHANGESET_CONTEXT", string(data))
		actual, err := readProjectContextFromEnv()
		require.NoError(t, err)
		require.Equal(t, ctx.Project, actual.Project)
		require.Equal(t, ctx.ProjectPath, actual.ProjectPath)
		require.Equal(t, ctx.ModulePath, actual.ModulePath)
	})

	t.Run("missing env var", func(t *testing.T) {
		orig := os.Getenv("CHANGESET_CONTEXT")
		defer os.Setenv("CHANGESET_CONTEXT", orig)

		os.Unsetenv("CHANGESET_CONTEXT")
		_, err := readProjectContextFromEnv()
		require.Error(t, err)
		require.Contains(t, err.Error(), "no context in CHANGESET_CONTEXT env var")
	})

	t.Run("invalid JSON", func(t *testing.T) {
		orig := os.Getenv("CHANGESET_CONTEXT")
		defer os.Setenv("CHANGESET_CONTEXT", orig)

		os.Setenv("CHANGESET_CONTEXT", "not valid json")
		_, err := readProjectContextFromEnv()
		require.Error(t, err)
		require.Contains(t, err.Error(), "failed to parse context JSON from env")
	})

	t.Run("missing project name", func(t *testing.T) {
		ctx := &models.ProjectContext{
			ProjectPath:    "/workspace/my-project",
			ModulePath:     "github.com/example/my-project",
			CurrentVersion: "1.0.0",
		}
		data, err := json.Marshal(ctx)
		require.NoError(t, err)

		orig := os.Getenv("CHANGESET_CONTEXT")
		defer os.Setenv("CHANGESET_CONTEXT", orig)

		os.Setenv("CHANGESET_CONTEXT", string(data))
		_, err = readProjectContextFromEnv()
		require.Error(t, err)
		require.Contains(t, err.Error(), "project name is required")
	})
}

func runWithStdin(input string, fn func() error) error {
	origStdin := os.Stdin
	defer func() { os.Stdin = origStdin }()

	r, w, err := os.Pipe()
	if err != nil {
		return err
	}
	defer r.Close()
	defer w.Close()

	if _, err := w.Write([]byte(input)); err != nil {
		return err
	}
	w.Close() // signal EOF

	os.Stdin = r
	return fn()
}

func runWithStdinViaTerminal(fn func() error) error {
	origStdin := os.Stdin
	defer func() { os.Stdin = origStdin }()

	nullDev, err := os.Open("/dev/null")
	if err != nil {
		return err
	}
	defer nullDev.Close()

	os.Stdin = nullDev
	return fn()
}
