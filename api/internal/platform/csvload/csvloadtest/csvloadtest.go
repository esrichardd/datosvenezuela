// Package csvloadtest holds helpers for testing the dataset loaders, in the
// spirit of net/http/httptest. Only tests import it.
package csvloadtest

import (
	"os"
	"path/filepath"
	"testing"
)

// WriteFiles writes files (file name → content) to a new temporary directory
// and returns its path. The directory is removed when the test ends.
func WriteFiles(t testing.TB, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o600); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}
	return dir
}
