package workspace

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/jakoblorz/go-changesets/internal/filesystem"
)

// GoEnv represents a minimal subset of go env output.
type GoEnv struct {
	GoWork string
	GoMod  string
}

// GoEnvReader abstracts go env lookups for testability.
type GoEnvReader interface {
	Read() (GoEnv, error)
}

type osGoEnvReader struct {
	fs filesystem.FileSystem
}

func newOSGoEnvReader(fs filesystem.FileSystem) GoEnvReader {
	return &osGoEnvReader{fs: fs}
}

func (r *osGoEnvReader) Read() (GoEnv, error) {
	cwd, err := r.fs.Getwd()
	if err != nil {
		return GoEnv{}, fmt.Errorf("failed to get working directory: %w", err)
	}

	cmd := exec.Command("go", "env", "-json", "GOWORK", "GOMOD")
	cmd.Dir = cwd

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		errMsg := strings.TrimSpace(stderr.String())
		if errMsg != "" {
			return GoEnv{}, fmt.Errorf("go env failed: %w: %s", err, errMsg)
		}
		return GoEnv{}, fmt.Errorf("go env failed: %w", err)
	}

	var raw struct {
		GoWork string `json:"GOWORK"`
		GoMod  string `json:"GOMOD"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &raw); err != nil {
		return GoEnv{}, fmt.Errorf("failed to parse go env output: %w", err)
	}

	return normalizeGoEnv(raw.GoWork, raw.GoMod), nil
}

// NewMockGoEnvReader creates a filesystem-backed Go env reader for tests.
func NewMockGoEnvReader(fs filesystem.FileSystem) GoEnvReader {
	return &mockGoEnvReader{fs: fs}
}

type mockGoEnvReader struct {
	fs filesystem.FileSystem
}

func (r *mockGoEnvReader) Read() (GoEnv, error) {
	cwd, err := r.fs.Getwd()
	if err != nil {
		return GoEnv{}, fmt.Errorf("failed to get working directory: %w", err)
	}

	if goWorkPath, found, err := findFileUp(r.fs, cwd, "go.work"); err != nil {
		return GoEnv{}, err
	} else if found {
		return GoEnv{GoWork: goWorkPath}, nil
	}

	if goModPath, found, err := findFileUp(r.fs, cwd, "go.mod"); err != nil {
		return GoEnv{}, err
	} else if found {
		return GoEnv{GoMod: goModPath}, nil
	}

	return GoEnv{}, nil
}

func normalizeGoEnv(goWork, goMod string) GoEnv {
	goWork = strings.TrimSpace(goWork)
	goMod = strings.TrimSpace(goMod)

	if strings.EqualFold(goWork, "off") {
		goWork = ""
	}

	if strings.EqualFold(goMod, os.DevNull) || strings.EqualFold(goMod, "NUL") {
		goMod = ""
	}

	if goWork != "" {
		goWork = filepath.Clean(goWork)
	}

	if goMod != "" {
		goMod = filepath.Clean(goMod)
	}

	return GoEnv{GoWork: goWork, GoMod: goMod}
}
