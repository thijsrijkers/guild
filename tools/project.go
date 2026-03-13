package tools

import (
    "io/fs"
    "path/filepath"
    "strings"
)

var ignoredDirs = map[string]bool{
    ".git": true, "node_modules": true, "vendor": true,
    "dist": true, "build": true, ".idea": true,
}

var ignoredExts = map[string]bool{
    ".exe": true, ".bin": true, ".png": true, ".jpg": true,
    ".zip": true, ".sum": true,
}

func BuildFileTree(root string) ([]string, error) {
    var files []string

    filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
        if d.IsDir() && ignoredDirs[d.Name()] {
            return filepath.SkipDir
        }
        if !d.IsDir() {
            ext := strings.ToLower(filepath.Ext(path))
            if !ignoredExts[ext] {
                files = append(files, path)
            }
        }
        return nil
    })

    return files, nil
}
