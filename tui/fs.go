package tui

import "os"

func readFileOS(path string) ([]byte, error) {
	return os.ReadFile(path)
}

func writeFileOS(path string, data []byte) error {
	return os.WriteFile(path, data, 0644)
}
