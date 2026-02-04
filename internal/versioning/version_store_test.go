package versioning

import (
	"strings"
	"testing"

	"github.com/jakoblorz/go-changesets/internal/filesystem"
	"github.com/jakoblorz/go-changesets/internal/models"
)

func TestPackageJSONVersionStore_ReadWrite(t *testing.T) {
	fs := filesystem.NewMockFileSystem()
	fs.AddFile("/workspace/package.json", []byte(`{
  "name": "app",
  "version": "0.1.0",
  "private": true
}`))

	store := NewPackageJSONVersionStore(fs)

	ver, err := store.Read("/workspace")
	if err != nil {
		t.Fatalf("Read() error = %v", err)
	}
	if got := ver.String(); got != "0.1.0" {
		t.Fatalf("unexpected version: %s", got)
	}

	newVer := ver.Bump(models.BumpMinor)
	if err := store.Write("/workspace", newVer); err != nil {
		t.Fatalf("Write() error = %v", err)
	}

	data, err := fs.ReadFile("/workspace/package.json")
	if err != nil {
		t.Fatalf("failed to read written package.json: %v", err)
	}

	if string(data) == "" || !strings.Contains(string(data), "\"version\": \"0.2.0\"") {
		t.Fatalf("expected package.json to contain new version, got: %s", string(data))
	}
}
