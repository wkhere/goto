package task

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"strings"
	"text/template"

	"github.com/akedrou/textdiff"
	"gitlab.com/wkhere/goto/core"
)

type Template struct {
	Name     string
	Template *template.Template
	Dest     string
	Mode     fs.FileMode
	core.Notifier
}

var _ core.Task = (*Template)(nil)

func MustParseTemplate(fsys fs.FS, path string) *template.Template {
	return template.Must(template.ParseFS(fsys, path))
}

func (t *Template) Desc() string { return t.Name }

func (t *Template) View(vars core.Vars) (changed bool, diff string,
	notify []core.Handler, err error) {

	dest, err := core.Eval("Template.Dest", t.Dest, vars)
	if err != nil {
		return false, "", nil, fmt.Errorf("eval dest: %w", err)
	}

	oldContent, _, err := readFile(dest)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return false, "", nil, fmt.Errorf("read old file: %w", err)
	}

	var newContent string
	err = t.evalTo(&newContent, nil, vars)
	if err != nil {
		return false, "", nil, fmt.Errorf("eval new content: %w", err)
	}

	diff = textdiff.Unified(dest, dest+"-new", oldContent, newContent)
	changed = len(diff) > 0
	if changed {
		notify = t.Notifier
	}
	return changed, diff, notify, nil

}

func (t *Template) Exec(vars core.Vars) (changed bool, diff string,
	notifies []core.Handler, err error) {

	dest, err := core.Eval("Template.Dest", t.Dest, vars)
	if err != nil {
		return false, "", nil, fmt.Errorf("eval dest: %w", err)
	}

	oldContent, oldMode, err := readFile(dest)
	var oldMissing bool
	if err != nil {
		if oldMissing = errors.Is(err, fs.ErrNotExist); !oldMissing {
			return false, "", nil, fmt.Errorf("read old file: %w", err)
		}
	}

	newMode := t.Mode
	if newMode == 0 && !oldMissing {
		newMode = oldMode
	}
	if newMode == 0 {
		return false, "", nil, fmt.Errorf("undefined task.Template.Mode %s", dest)
	}
	newPath, err := tmpPath(dest)
	if err != nil {
		return false, "", nil, fmt.Errorf("create tmp path: %w", err)
	}
	newFile, err := os.OpenFile(newPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, newMode)
	if err != nil {
		return false, "", nil, fmt.Errorf("create tmp file: %w", err)
	}

	var newContent string
	err = t.evalTo(&newContent, newFile, vars)
	if err != nil {
		newFile.Close()
		return false, "", nil, fmt.Errorf("eval new content: %w", err)
	}

	diff = textdiff.Unified(dest, dest+"-new", oldContent, newContent)
	changed = len(diff) > 0

	err = newFile.Close()
	if err != nil {
		return changed, diff, nil, fmt.Errorf("close tmp file: %w", err)
	}

	if !changed {
		err = os.Remove(newPath)
		if err != nil {
			err = fmt.Errorf("remove tmp file: %w", err)
		}
		return changed, diff, nil, err
	}

	if !oldMissing {
		err = os.Rename(dest, dest+"~")
		if err != nil {
			return changed, diff, nil, fmt.Errorf("backup dest: %w", err)
		}
	}
	err = os.Rename(newPath, dest)
	if err != nil {
		return changed, diff, nil, fmt.Errorf("overwrite dest: %w", err)
	}

	return changed, diff, t.Notifier, nil

}

func (t *Template) evalTo(p *string, tee io.Writer, vars core.Vars) error {
	b := new(strings.Builder)

	var w io.Writer = b
	if tee != nil {
		w = io.MultiWriter(b, tee)
	}

	err := t.Template.Execute(w, vars)
	*p = b.String()
	return err
}

func readFile(path string) (string, fs.FileMode, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", 0, err
	}
	defer f.Close()

	st, err := f.Stat()
	if err != nil {
		return "", 0, err
	}

	b := new(strings.Builder)

	_, err = io.Copy(b, f)
	return b.String(), st.Mode(), err
}
