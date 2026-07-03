package task

import (
	"fmt"
	"os"
	"path/filepath"
)

func tmpPath(path string) (string, error) {
	d, f := filepath.Split(path)
	if d == "" {
		return "", fmt.Errorf("path %q has empty dir part", path)
	}
	if f == "" {
		return "", fmt.Errorf("path %q has empty file part", path)
	}
	return filepath.Join(d, fmt.Sprintf(".%s.goto-%d", f, os.Getpid())), nil

}
