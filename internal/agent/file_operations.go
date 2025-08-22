package agent

import (
	"os"
	"path/filepath"
	"strings"
)

// listDir recursively lists directory contents
func listDir(path string, depth int, lines *[]string, basePath string) error {
	if depth < 0 {
		return nil
	}

	entries, err := os.ReadDir(path)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		fullPath := filepath.Join(path, entry.Name())
		
		// Calculate relative path from base
		relPath, err := filepath.Rel(basePath, fullPath)
		if err != nil {
			relPath = fullPath
		}

		// Add indentation based on depth level
		indent := strings.Repeat("  ", len(strings.Split(relPath, string(filepath.Separator)))-1)
		
		if entry.IsDir() {
			*lines = append(*lines, indent+entry.Name()+"/")
			// Recursively list subdirectories if depth allows
			if depth > 0 {
				if err := listDir(fullPath, depth-1, lines, basePath); err != nil {
					// Continue on error, just note it
					*lines = append(*lines, indent+"  (error reading directory)")
				}
			}
		} else {
			*lines = append(*lines, indent+entry.Name())
		}
	}

	return nil
}