package prompt

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

var ignoredDirs = map[string]bool{
	".git": true, "node_modules": true, "vendor": true,
	"dist": true, "build": true, ".idea": true, ".vscode": true,
}

var ignoredExts = map[string]bool{
	".exe": true, ".bin": true, ".png": true, ".jpg": true,
	".jpeg": true, ".gif": true, ".zip": true, ".tar": true,
	".gz": true, ".sum": true, ".lock": true,
}

// FileEntry represents a single file in the project.
type FileEntry struct {
	Path    string
	RelPath string
}

// BuildFileList walks the given root and returns all relevant files.
func BuildFileList(root string) ([]FileEntry, error) {
	var entries []FileEntry

	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			if ignoredDirs[d.Name()] {
				return filepath.SkipDir
			}
			return nil
		}

		ext := strings.ToLower(filepath.Ext(path))
		if ignoredExts[ext] {
			return nil
		}

		rel, err := filepath.Rel(root, path)
		if err != nil {
			rel = path
		}

		entries = append(entries, FileEntry{Path: path, RelPath: rel})
		return nil
	})

	return entries, err
}

// Build constructs the full system prompt, injecting the file tree.
func Build(entries []FileEntry) string {
	var sb strings.Builder

	sb.WriteString("You are Oda, an AI coding assistant running inside a terminal.\n")
	sb.WriteString("You are working inside the user's project directory.\n\n")

	sb.WriteString("The project contains the following files:\n")
	for _, e := range entries {
		sb.WriteString(fmt.Sprintf("  %s\n", e.RelPath))
	}

	sb.WriteString(`
When you need to read a file to answer the user's question, respond with:
<action>{"type": "read_file", "path": "relative/path/to/file.go"}</action>

When you want to apply a targeted edit to a file, respond with:
<action>{"type": "replace_in_file", "path": "file.go", "old": "existing code", "new": "replacement code"}</action>

When you want to write an entire file, respond with:
<action>{"type": "write_file", "path": "file.go", "content": "full file content"}</action>

Rules:
- Only read files that are genuinely relevant to the question.
- Always explain what you are doing before emitting an action.
- After reading a file, answer directly without asking for permission.
- Keep responses concise and focused on code.
`)

	return sb.String()
}

// ReadFile reads a file and returns its content as a string.
func ReadFile(path string) (string, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(content), nil
}
