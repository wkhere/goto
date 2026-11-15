package task

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/wkhere/goto/core"
)

func TestSimpleTemplate(t *testing.T) {
	const fooTemplPath = "foo.t"

	srcfs := fstest.MapFS{
		fooTemplPath: {Data: []byte("foo should be equal to {{.Foo}}\n")},
	}

	dest, err := os.CreateTemp(t.TempDir(), "*")
	if err != nil {
		t.Errorf("unexpected create-temp-dest error: %v", err)
		return
	}
	destSt, err := dest.Stat()
	if err != nil {
		t.Errorf("unexpected stat-temp-dest error: %v", err)
		return
	}

	_, err = io.WriteString(dest, "foo should be equal to foo\n")
	if err != nil {
		t.Errorf("unexpected write-temp-dest error: %v", err)
		return
	}
	err = dest.Close()
	if err != nil {
		t.Error(err)
		return
	}

	task := &Template{
		Name:     "test template foo",
		Template: MustParseTemplate(srcfs, fooTemplPath),
		Dest:     `{{.MyDest}}`,
		Mode:     destSt.Mode(),
	}

	vars := core.Vars{
		"Foo":    "foo",
		"MyDest": dest.Name(),
	}

	varsContentChanged, err := vars.Eval(nil, core.Vars{
		"Foo": "quux",
	})
	if err != nil {
		t.Errorf("unexpected eval error: %v", err)
		return
	}

	varsDestNotExists := func(t *testing.T) core.Vars {
		t.Helper()
		vv, err := vars.Eval(nil, core.Vars{
			"MyDest": filepath.Join(t.TempDir(), "nonexistent"),
		})
		if err != nil {
			t.Fatalf("unexpected eval error: %v", err)
		}
		return vv
	}

	t.Run("Desc", func(t *testing.T) {
		if task.Desc() == "" {
			t.Errorf("empty task desc")
		}
	})

	t.Run("View-unchanged", func(t *testing.T) {
		changed, diff, _, err := task.View(vars)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if changed {
			t.Error("expected changed=false")
		}
		if diff != "" {
			t.Errorf("expected empty diff, have:\n%s", diff)
		}

	})

	t.Run("View-changed", func(t *testing.T) {
		wdiff := "-foo should be equal to foo\n+foo should be equal to quux\n"

		changed, diff, _, err := task.View(varsContentChanged)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if !changed {
			t.Error("expected changed=true")
		}
		if !strings.Contains(diff, wdiff) {
			t.Errorf("diff mismatch, have:\n%s", diff)
		}
	})

	t.Run("View-mode-changed", func(t *testing.T) {
		task2 := *task
		task2.Mode = 0400
		wdiff := fmt.Sprintf("--- mode: 0%o\n+++ mode: 0%o\n", task.Mode, task2.Mode)

		changed, diff, _, err := task2.View(vars)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if !changed {
			t.Error("expected changed=true")
		}
		if !strings.Contains(diff, wdiff) {
			t.Errorf("diff mismatch, have:\n%s", diff)
		}
	})

	t.Run("View-mode-zero", func(t *testing.T) {
		task2 := *task
		task2.Mode = 0

		changed, diff, _, err := task2.View(vars)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if changed {
			t.Error("expected changed=false")
		}
		if diff != "" {
			t.Errorf("expected empty diff, have:\n%s", diff)
		}
	})

	t.Run("View-nonexistent-dest", func(t *testing.T) {
		changed, diff, _, err := task.View(varsDestNotExists(t))
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if !changed {
			t.Error("expected changed=true")
		}
		if diff == "" {
			t.Error("expected non-empty diff")
		}
	})

	t.Run("View-nonexistent-dest-mode-zero", func(t *testing.T) {
		task2 := *task
		task2.Mode = 0
		wdiff := `+foo should be equal to foo`

		changed, diff, _, err := task2.View(varsDestNotExists(t))
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if !changed {
			t.Error("expected changed=true")
		}
		if !strings.Contains(diff, wdiff) {
			t.Errorf("diff mismatch, have:\n%s", diff)
		}
	})

	t.Run("View-broken-dest", func(t *testing.T) {
		task2 := *task
		task2.Dest = `{{.Bad}}`
		werr := "undefined var Bad"

		changed, diff, _, err := task2.View(vars)
		if err == nil || !strings.Contains(err.Error(), werr) {
			t.Errorf("expected error with `%s`; have: %v", werr, err)
		}
		if changed {
			t.Error("expected changed=false")
		}
		if diff != "" {
			t.Error("expected empty diff")
		}
	})

	t.Run("View-empty-dest-dir", func(t *testing.T) {
		task2 := *task
		task2.Dest = `foo`
		werr := `path "foo" has empty dir part`
		// todo: impl should handle this differently

		changed, diff, _, err := task2.View(vars)
		if err == nil || !strings.Contains(err.Error(), werr) {
			t.Errorf("expected error with `%s`; have: %v", werr, err)
		}
		if changed {
			t.Error("expected changed=false")
		}
		if diff != "" {
			t.Error("expected empty diff")
		}
	})

	t.Run("View-empty-dest-filename", func(t *testing.T) {
		task2 := *task
		task2.Dest = `foo/`
		werr := `path "foo/" has empty file part`
		// todo: impl should handle this differently

		changed, diff, _, err := task2.View(vars)
		if err == nil || !strings.Contains(err.Error(), werr) {
			t.Errorf("expected error with `%s`; have: %v", werr, err)
		}
		if changed {
			t.Error("expected changed=false")
		}
		if diff != "" {
			t.Error("expected empty diff")
		}
	})

	t.Run("Exec-unchanged", func(t *testing.T) {
		changed, diff, _, err := task.Exec(vars)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if changed {
			t.Error("expected changed=false")
		}
		if diff != "" {
			t.Errorf("expected empty diff, have:\n%s", diff)
		}
	})

	t.Run("Exec-changed", func(t *testing.T) {
		wdiff := "-foo should be equal to foo\n+foo should be equal to quux\n"

		changed, diff, _, err := task.Exec(varsContentChanged)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if !changed {
			t.Error("expected changed=true")
		}
		if !strings.Contains(diff, wdiff) {
			t.Errorf("diff mismatch, have:\n%s", diff)
		}
	})

	t.Run("Exec-mode-changed", func(t *testing.T) {
		task2 := *task
		task2.Mode = 0400
		wdiff := fmt.Sprintf("--- mode: 0%o\n+++ mode: 0%o\n", task.Mode, task2.Mode)

		changed, diff, _, err := task2.Exec(vars)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if !changed {
			t.Error("expected changed=true")
		}
		if !strings.Contains(diff, wdiff) {
			t.Errorf("diff mismatch, have:\n%s", diff)
		}
	})

	t.Run("Exec-mode-zero", func(t *testing.T) {
		task2 := *task
		task2.Mode = 0

		changed, diff, _, err := task2.Exec(vars)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if changed {
			t.Error("expected changed=false")
		}
		if diff != "" {
			t.Errorf("expected empty diff, have:\n%s", diff)
		}
	})

	t.Run("Exec-nonexistent-dest", func(t *testing.T) {
		wdiff := `+foo should be equal to foo`

		changed, diff, _, err := task.Exec(varsDestNotExists(t))
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if !changed {
			t.Error("expected changed=true")
		}
		if !strings.Contains(diff, wdiff) {
			t.Errorf("diff mismatch, have:\n%s", diff)
		}
	})

	t.Run("Exec-nonexistent-dest-mode-zero", func(t *testing.T) {
		task2 := *task
		task2.Mode = 0
		wdiff := `+foo should be equal to foo`

		changed, diff, _, err := task2.Exec(varsDestNotExists(t))
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if !changed {
			t.Error("expected changed=true")
		}
		if !strings.Contains(diff, wdiff) {
			t.Errorf("diff mismatch, have:\n%s", diff)
		}
	})

	t.Run("Exec-broken-dest", func(t *testing.T) {
		task2 := *task
		task2.Dest = `{{.Bad}}`
		werr := "undefined var Bad"

		changed, diff, _, err := task2.Exec(vars)
		if err == nil || !strings.Contains(err.Error(), werr) {
			t.Errorf("expected error with `%s`; have: %v", werr, err)
		}
		if changed {
			t.Error("expected changed=false")
		}
		if diff != "" {
			t.Error("expected empty diff")
		}
	})

	t.Run("Exec-empty-dest-dir", func(t *testing.T) {
		task2 := *task
		task2.Dest = `foo`
		werr := `path "foo" has empty dir part`
		// todo: impl should handle this differently

		changed, diff, _, err := task2.Exec(vars)
		if err == nil || !strings.Contains(err.Error(), werr) {
			t.Errorf("expected error with `%s`; have: %v", werr, err)
		}
		if changed {
			t.Error("expected changed=false")
		}
		if diff != "" {
			t.Error("expected empty diff")
		}
	})

	t.Run("Exec-empty-dest-filename", func(t *testing.T) {
		task2 := *task
		task2.Dest = `foo/`
		werr := `path "foo/" has empty file part`
		// todo: impl should handle this differently

		changed, diff, _, err := task2.Exec(vars)
		if err == nil || !strings.Contains(err.Error(), werr) {
			t.Errorf("expected error with `%s`; have: %v", werr, err)
		}
		if changed {
			t.Error("expected changed=false")
		}
		if diff != "" {
			t.Error("expected empty diff")
		}
	})
}

func TestTemplateTmpCleanup(t *testing.T) {
	srcfs := fstest.MapFS{
		"good.t": {Data: []byte("foo should be equal to {{.Foo}}\n")},
		"bad.t":  {Data: []byte("{{.Foo.NoSuchField}}\n")},
	}
	dest := filepath.Join(t.TempDir(), "dest")
	vars := core.Vars{"Foo": "foo"}

	bad := &Template{
		Name:     "bad template",
		Template: MustParseTemplate(srcfs, "bad.t"),
		Dest:     dest,
	}
	good := &Template{
		Name:     "good template",
		Template: MustParseTemplate(srcfs, "good.t"),
		Dest:     dest,
	}

	_, _, _, err := bad.Exec(vars)
	if err == nil {
		t.Fatal("expected eval error")
	}

	tmp, err := tmpPath(dest)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, err := os.Lstat(tmp); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("tmp file left behind after failed exec: %s", tmp)
	}

	changed, _, _, err := good.Exec(vars)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if !changed {
		t.Error("expected changed=true")
	}
}
