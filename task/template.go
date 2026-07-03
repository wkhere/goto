package task

import (
	"fmt"
	"io"
	"os"
	"strings"
	"text/template"

	"github.com/akedrou/textdiff"
	"gitlab.com/wkhere/goto/core"
)

type Template struct {
	Name       string
	SourceName string
	Content    string
	Dest       string
}

func (t Template) Desc() string { return t.Name }

func (t Template) View(vars core.Vars) (changed bool, diff string, err error) {
	dest, err := core.Eval("template.Dest", t.Dest, vars)
	if err != nil {
		return false, "", fmt.Errorf("eval dest: %w", err)
	}

	oldContent, _, err := readToBuf(dest)
	if err != nil {
		return false, "", fmt.Errorf("read dest: %w", err)
	}

	newContent, err := t.evalToBuf(vars, nil)
	if err != nil {
		return false, "", fmt.Errorf("eval new content: %w", err)
	}

	diff = textdiff.Unified(dest, dest+"-new", oldContent.String(), newContent.String())
	changed = len(diff) > 0
	return changed, diff, nil

}

func (t Template) Exec(vars core.Vars) (changed bool, diff string, err error) {
	dest, err := core.Eval("template.Dest", t.Dest, vars)
	if err != nil {
		return false, "", fmt.Errorf("eval dest: %w", err)
	}

	oldContent, fileMode, err := readToBuf(dest)
	if err != nil {
		return false, "", fmt.Errorf("read dest: %w", err)
	}

	newPath, err := tmpPath(dest)
	if err != nil {
		return false, "", fmt.Errorf("create tmp path: %w", err)
	}
	newFile, err := os.OpenFile(newPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, fileMode)
	if err != nil {
		return false, "", fmt.Errorf("create tmp file: %w", err)
	}

	newContent, err := t.evalToBuf(vars, newFile)
	if err != nil {
		newFile.Close()
		return false, "", fmt.Errorf("eval new content: %w", err)
	}

	diff = textdiff.Unified(dest, dest+"-new", oldContent.String(), newContent.String())
	changed = len(diff) > 0

	err = newFile.Close()
	if err != nil {
		return changed, diff, fmt.Errorf("close tmp file: %w", err)
	}

	if !changed {
		err = os.Remove(newPath)
		if err != nil {
			err = fmt.Errorf("remove tmp file: %w", err)
		}
		return changed, diff, err
	}

	err = os.Rename(dest, dest+"~")
	if err != nil {
		return changed, diff, fmt.Errorf("backup dest: %w", err)
	}
	err = os.Rename(newPath, dest)
	if err != nil {
		return changed, diff, fmt.Errorf("overwrite dest: %w", err)
	}

	return changed, diff, nil

}

func (t Template) guessSize() int { return ((len(t.Content)+100)/4096 + 1) * 4096 }

func (t Template) evalToBuf(vars core.Vars, tee io.Writer) (*strings.Builder, error) {
	llt, err := template.New(t.SourceName).Parse(t.Content)
	if err != nil {
		return nil, err
	}

	b := new(strings.Builder)
	b.Grow(t.guessSize())

	var w io.Writer = b
	if tee != nil {
		w = io.MultiWriter(b, tee)
	}

	err = llt.Execute(w, vars)
	return b, err
}

func readToBuf(path string) (*strings.Builder, os.FileMode, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, 0, err
	}
	defer f.Close()

	st, err := f.Stat()
	if err != nil {
		return nil, 0, err
	}

	b := new(strings.Builder)
	b.Grow(int(st.Size()))

	_, err = io.Copy(b, f)
	return b, st.Mode(), err
}
