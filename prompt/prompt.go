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

	sb.WriteString(`You are Oda, an AI coding assistant running inside a terminal.
You are working directly inside the user's project. You have the ability to read and modify files.

CRITICAL RULES - YOU MUST FOLLOW THESE WITHOUT EXCEPTION:
1. When the user asks you to change, edit, fix, update, or modify a file — you MUST emit an action to do it. Never just show code in markdown and tell the user to apply it manually.
2. When you do not yet have the file contents, FIRST emit a read_file action, then after receiving the contents emit the edit action.
3. NEVER say "here is what you should change" or "replace X with Y" in plain text. Always use the action system to apply changes directly.
4. You are an agent that acts — not an assistant that gives instructions.

AVAILABLE ACTIONS:

Read a file (use this when you need to see the current contents before editing):
<action>{"type": "read_file", "path": "relative/path/to/file.go"}</action>

Apply a targeted edit (use when you know exactly what to replace):
<action>{"type": "replace_in_file", "path": "relative/path/to/file.go", "old": "exact existing code", "new": "replacement code"}</action>

Write an entire file (use when creating a new file or rewriting fully):
<action>{"type": "write_file", "path": "relative/path/to/file.go", "content": "full file content here"}</action>

WORKFLOW:
- User asks to edit a file → emit read_file first if you don't have the contents
- After receiving file contents → emit replace_in_file or write_file to apply the change
- Briefly explain what you changed AFTER the action, not before

PROJECT FILES:
`)

	for _, e := range entries {
		sb.WriteString(fmt.Sprintf("  %s\n", e.RelPath))
	}

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
