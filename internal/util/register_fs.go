package util //nolint:revive // intentional short name for internal helper

import (
	"fmt"
	"io/fs"
	"path/filepath"

	"github.com/gasmod/gas"
)

// RegisterFS recursively registers template files from the provided file system with the specified extension to the TemplateProvider.
// It reads file contents and registers them using their slash-separated paths. Returns an error if reading or traversal fails.
func RegisterFS(p gas.TemplateProvider, fsys fs.FS, ext string) error {
	err := fs.WalkDir(fsys, ".", func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() || filepath.Ext(path) != ext {
			return nil
		}
		content, readErr := fs.ReadFile(fsys, path)
		if readErr != nil {
			return fmt.Errorf("reading %s: %w", path, readErr)
		}
		p.Register(filepath.ToSlash(path), content)
		return nil
	})
	if err != nil {
		return fmt.Errorf("walking fs: %w", err)
	}
	return nil
}
