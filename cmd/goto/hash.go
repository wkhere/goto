package main

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
	"path/filepath"
)

func hashFromLocalWD() (string, error) {
	h := sha256.New()

	s, err := os.Hostname()
	if err != nil {
		return "", err
	}
	io.WriteString(h, s)
	io.WriteString(h, ":")

	s, err = os.Getwd()
	if err != nil {
		return "", err
	}
	s, err = realpath(s)
	if err != nil {
		return "", err
	}
	io.WriteString(h, s)

	return hex.EncodeToString(h.Sum(nil)[:6]), nil
}

func realpath(path string) (_ string, err error) {
	path, err = filepath.Abs(path)
	if err != nil {
		return "", err
	}
	return filepath.EvalSymlinks(path)
}
