package agent

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// performFileSearch performs the actual file search with regex
func performFileSearch(pattern, searchPath string, fileTypes []string, caseSensitive bool, maxResults int, workingDir string) ([]SearchResult, error) {
	// Compile regex pattern
	var regex *regexp.Regexp
	var err error

	if caseSensitive {
		regex, err = regexp.Compile(pattern)
	} else {
		regex, err = regexp.Compile("(?i)" + pattern)
	}

	if err != nil {
		return nil, fmt.Errorf("invalid regex pattern: %w", err)
	}

	// Create file type map for quick lookup
	var typeMap map[string]bool
	searchAllFiles := len(fileTypes) == 0

	if !searchAllFiles {
		typeMap = make(map[string]bool)
		for _, ext := range fileTypes {
			typeMap["."+ext] = true
		}
	}

	var results []SearchResult

	// Use working directory if provided
	baseDir := workingDir
	if baseDir == "" {
		if wd, err := os.Getwd(); err == nil {
			baseDir = wd
		} else {
			baseDir = "."
		}
	}

	base := filepath.Join(baseDir, searchPath)
	if !filepath.IsAbs(base) {
		abs, err := filepath.Abs(base)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve path: %w", err)
		}
		base = abs
	}

	err = filepath.Walk(base, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip files we can't access
		}

		if info.IsDir() {
			// Skip common heavy directories
			dirName := filepath.Base(path)
			if dirName == "node_modules" || dirName == ".git" || dirName == ".venv" ||
				dirName == "__pycache__" || dirName == "dist" || dirName == "build" ||
				dirName == "target" || dirName == "coverage" {
				return filepath.SkipDir
			}
			return nil
		}

		// Check file extension if we have type restrictions
		if !searchAllFiles {
			ext := filepath.Ext(path)
			if !typeMap[ext] {
				return nil
			}
		}

		// Skip large files
		if info.Size() > maxFileSize {
			return nil
		}

		// Read and search file
		content, err := os.ReadFile(path)
		if err != nil {
			return nil // Skip files we can't read
		}

		lines := strings.Split(string(content), "\n")
		for lineNum, line := range lines {
			if regex.MatchString(line) {
				relPath, _ := filepath.Rel(base, path)
				results = append(results, SearchResult{
					File:    relPath,
					LineNum: lineNum + 1,
					Line:    line,
				})

				if len(results) >= maxResults {
					return fmt.Errorf("max results reached")
				}
			}
		}

		return nil
	})

	if err != nil && err.Error() != "max results reached" {
		return results, err
	}

	return results, nil
}

