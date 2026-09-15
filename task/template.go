package task

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"strings"
	"text/template"

	"codeberg.org/wkhere/goto/core"
	"github.com/akedrou/textdiff"
)

type Template struct {
	Name     string
	Template *template.Template
	Dest     string
	Mode     fs.FileMode
	Vars     core.Vars
	core.Notifier
}

var _ core.Task = (*Template)(nil)

func MustParseTemplate(fsys fs.FS, path string) *template.Template {
	return template.Must(template.ParseFS(fsys, path))
}

func (t *Template) Desc() string { return t.Name }

func (t *Template) View(vars core.Vars) (changed bool, diff string, notify []core.Handler,
	err error) {

	vars, dest, err := vars.EvalWithExpr(t.Dest, core.Funcs, t.Vars)
	if err != nil {
		return false, "", nil, fmt.Errorf("eval dest: %w", err)
	}

	oldContent, oldMode, oldExists, err := readFile(dest)
	if err != nil {
		return false, "", nil, fmt.Errorf("read old file: %w", err)
	}

	newPath, err := tmpPath(dest)
	if err != nil {
		return false, "", nil, fmt.Errorf("create tmp path: %w", err)
	}
	var newContent string
	err = t.evalTo(&newContent, nil, vars)
	if err != nil {
		return false, "", nil, fmt.Errorf("eval new content: %w", err)
	}

	diff = textdiff.Unified(dest, newPath, oldContent, newContent)
	changed = len(diff) > 0

	if oldExists {
		modeChanged, diff2 := modeDiff(oldMode, t.Mode)
		changed = changed || modeChanged
		if modeChanged {
			diff += diff2
		}
	}

	if changed {
		notify = t.Notifier
	}
	return changed, diff, notify, nil
}

func (t *Template) Exec(vars core.Vars) (changed bool, diff string, notify []core.Handler,
	err error) {

	vars, dest, err := vars.EvalWithExpr(t.Dest, core.Funcs, t.Vars)
	if err != nil {
		return false, "", nil, fmt.Errorf("eval dest: %w", err)
	}

	oldContent, oldMode, oldExists, err := readFile(dest)
	if err != nil {
		return false, "", nil, fmt.Errorf("read old file: %w", err)
	}

	newMode := t.Mode
	if newMode == 000 {
		if oldExists {
			newMode = oldMode
		} else {
			newMode = 0666
		}
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

	diff = textdiff.Unified(dest, newPath, oldContent, newContent)
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
		goto compareMode
	}

	if oldExists {
		err = os.Rename(dest, dest+"~")
		if err != nil {
			return changed, diff, nil, fmt.Errorf("backup dest: %w", err)
		}
	}
	err = os.Rename(newPath, dest)
	if err != nil {
		// todo: try to rename back the original from backup
		return changed, diff, nil, fmt.Errorf("overwrite dest: %w", err)
	}

compareMode:
	if oldExists {
		modeChanged, diff2 := modeDiff(oldMode, t.Mode)
		changed = changed || modeChanged
		if modeChanged {
			err = os.Chmod(dest, t.Mode)
			if err != nil {
				err = fmt.Errorf("change dest file mode: %w", err)
				return
			}
			diff += diff2
		}
	}

	if changed {
		notify = t.Notifier
	}
	return changed, diff, notify, nil
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

func readFile(path string) (content string, mode fs.FileMode, exists bool, err error) {
	f, err := os.Open(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return "", 0, false, nil
		}
		return "", 0, true, err
	}
	defer f.Close()

	st, err := f.Stat()
	if err != nil {
		return "", 0, true, err
	}

	b := new(strings.Builder)

	_, err = io.Copy(b, f)
	return b.String(), st.Mode(), true, err
}
