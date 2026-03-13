package tools

import (
    "os"
    "strings"
)

func ReadFile(path string) (string, error) {
    content, err := os.ReadFile(path)
    if err != nil {
        return "", err
    }
    return string(content), nil
}

func WriteFile(path, content string) error {
    return os.WriteFile(path, []byte(content), 0644)
}

func ReplaceInFile(path, oldStr, newStr string) error {
    content, err := os.ReadFile(path)
    if err != nil {
        return err
    }
    updated := strings.ReplaceAll(string(content), oldStr, newStr)
    return os.WriteFile(path, []byte(updated), 0644)
}
