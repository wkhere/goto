package task

import (
	"io"
	"os"
	//"path/filepath"
	//"strings"
	"testing"
	"testing/fstest"

	"github.com/wkhere/goto/core"
)

func TestSimpleCopy(t *testing.T) {
	srcfs := fstest.MapFS{
		"foo": {Data: []byte("foo should be equal to foo\n")},
	}

	dest, err := os.CreateTemp(t.TempDir(), "*")
	if err != nil {
		t.Errorf("unexpected create-temp-dest error: %v", err)
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

	task := &Copy{
		Name: "test copy",
		Src:  "foo",
		Dest: `{{.MyDest}}`,
		//Mode: 0755,
	}

	vars := core.Vars{"Foo": "foo", "MyDest": dest.Name()}

	_ = srcfs
	_ = task
	_ = vars
}
