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
	hw := &errwriter{Writer: h}

	s, err := os.Hostname()
	if err != nil {
		return "", err
	}
	io.WriteString(hw, s)
	io.WriteString(hw, ":")

	s, err = os.Getwd()
	if err != nil {
		return "", err
	}
	s, err = realpath(s)
	if err != nil {
		return "", err
	}
	io.WriteString(hw, s)

	if hw.err != nil {
		return "", hw.err
	}
	return hex.EncodeToString(h.Sum(nil)[:6]), nil
}

func realpath(path string) (_ string, err error) {
	path, err = filepath.Abs(path)
	if err != nil {
		return "", err
	}
	return filepath.EvalSymlinks(path)
}

type errwriter struct {
	io.Writer
	err error
}

func (ew *errwriter) Write(p []byte) (n int, _ error) {
	if ew.err != nil {
		return 0, ew.err
	}
	n, ew.err = ew.Writer.Write(p)
	return n, ew.err
}
