package task

import (
	"encoding/hex"
	"errors"
	"fmt"
	"hash"
	"hash/fnv"
	"io"
	"io/fs"
	"math"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"codeberg.org/wkhere/goto/core"
)

type Copy struct {
	Name string
	Src  string
	Dest string
	Mode fs.FileMode
	Vars core.Vars
	core.Notifier
}

var (
	_ core.Task     = (*Copy)(nil)
	_ core.Transfer = (*Copy)(nil)
)

func (c *Copy) Desc() string { return c.Name }

func (c *Copy) hasMode() bool { return c.Mode != 000 }

func (c *Copy) RemotePrep(vars core.Vars) (stats string, err error) {

	vars, dest, err := vars.EvalWithExpr(c.Dest, core.Funcs, c.Vars)
	if err != nil {
		return "", fmt.Errorf("eval dest: %w", err)
	}
	transfer, err := tmpPath(dest)
	if err != nil {
		return "", fmt.Errorf("transfer path: %w", err)
	}

	err = os.Link(dest, transfer)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			if _, err := os.Stat(transfer); err == nil {
				// no dest, but transfer file exists - leave it
				return "prep: no previous file, transfer file found", nil
			}
			mode := c.Mode
			if !c.hasMode() {
				mode = 0666
			}
			// no dest no transfer file, touch the latter
			f, err := os.OpenFile(transfer, os.O_WRONLY|os.O_CREATE, mode)
			if err != nil {
				return "", fmt.Errorf("touch the transfer file: %w", err)
			}
			err = f.Close()
			if err != nil {
				return "", fmt.Errorf("touch&close the transfer file: %w", err)
			}
			return "prep: no previous file, transfer file touched", nil
		}
		if errors.Is(err, fs.ErrExist) {
			// transfer file exists - leave it
			return "prep: transfer file found, leaving", nil
		}
		// other link error
		return "", fmt.Errorf("prep/hardlink the transfer file: %w", err)
	}
	// transfer file linked
	return "prep/hardlink the transfer file: ok", nil
}

func (c *Copy) DoTransfer(vars core.Vars) (stats string, err error) {
	// note: this is run locally

	vars, src, err := vars.EvalWithExpr(c.Src, core.Funcs, c.Vars)
	if err != nil {
		return "", fmt.Errorf("eval src: %w", err)
	}
	dest, err := vars.EvalExpr(c.Dest, core.Funcs)
	if err != nil {
		return "", fmt.Errorf("eval dest: %w", err)
	}
	dest, err = tmpPath(dest)
	if err != nil {
		return "", fmt.Errorf("transfer path: %w", err)
	}

	host := vars[core.MetavarRemote]
	if host == "" {
		return "", fmt.Errorf("%s metavar not set", core.MetavarRemote)
	}

	cmd := exec.Command("rsync", "-vc", src, fmt.Sprintf("%s:%s", host, dest))

	buf := new(strings.Builder)
	cmd.Stdout = buf
	cmd.Stderr = os.Stderr

	err = cmd.Run()
	if err != nil {
		return "", fmt.Errorf("rsync: %w", err)
	}
	return parseRsyncStats(buf.String())
}

func (c *Copy) View(vars core.Vars) (changed bool, diff string, notify []core.Handler, err error) {

	vars, dest, err := vars.EvalWithExpr(c.Dest, core.Funcs, c.Vars)
	if err != nil {
		err = fmt.Errorf("eval dest: %w", err)
		return
	}
	transferred, err := tmpPath(dest)
	if err != nil {
		err = fmt.Errorf("transferred path: %w", err)
		return
	}

	missing, changed, diff, err := bindiffTransferred(dest, transferred)
	if err != nil {
		err = fmt.Errorf("bindiff: %w", err)
		return
	}

	if !missing {
		// perhaps I could leave removal of the transfer file also if sizes differ too much,
		// not only when dest is missing
		err = os.Remove(transferred)
		if err != nil {
			warnf("remove transferred file: %v", err)
		}

		var st os.FileInfo
		st, err = os.Stat(dest)
		if err != nil {
			err = fmt.Errorf("compare dest file mode: %w", err)
			return
		}
		modeChanged, diff2 := modeDiff(st.Mode(), c.Mode)
		changed = changed || modeChanged
		if modeChanged {
			diff += diff2
		}
	}

	if changed {
		notify = c.Notifier
	}
	return changed, diff, notify, nil
}

func (c *Copy) Exec(vars core.Vars) (changed bool, diff string, notify []core.Handler, err error) {

	vars, dest, err := vars.EvalWithExpr(c.Dest, core.Funcs, c.Vars)
	if err != nil {
		err = fmt.Errorf("eval dest: %w", err)
		return
	}
	transferred, err := tmpPath(dest)
	if err != nil {
		err = fmt.Errorf("transferred path: %w", err)
		return
	}

	missing, changed, diff, err := bindiffTransferred(dest, transferred)
	if err != nil {
		err = fmt.Errorf("bindiff: %w", err)
		return
	}

	if !changed {
		if !missing {
			err := os.Remove(transferred)
			if err != nil {
				warnf("remove transferred file: %v", err)
			}
		}
		goto compareMode
	}

	if !missing {
		err = os.Rename(dest, dest+"~")
		if err != nil {
			return true, diff, nil, fmt.Errorf("backup dest: %w", err)
		}
	}
	err = os.Rename(transferred, dest)
	if err != nil {
		// todo: try to rename back the original from backup
		err = fmt.Errorf("rename to dest: %w", err)
		return
	}

compareMode:
	st, err := os.Stat(dest)
	if err != nil {
		err = fmt.Errorf("compare dest file mode: %w", err)
		return
	}
	modeChanged, diff2 := modeDiff(st.Mode(), c.Mode)
	changed = changed || modeChanged
	if modeChanged {
		err = os.Chmod(dest, c.Mode)
		if err != nil {
			err = fmt.Errorf("change dest file mode: %w", err)
			return
		}
		diff += diff2
	}

	if changed {
		notify = c.Notifier
	}
	return changed, diff, notify, nil
}

func bindiffTransferred(dest, transferred string) (missing, changed bool, diff string, err error) {

	f, err := os.Open(dest)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			missing = true
			f, err = os.Open(os.DevNull)
			if err != nil {
				panic(err)
			}
		} else {
			err = fmt.Errorf("dest file: %w", err)
			return
		}
	}
	defer f.Close()

	t, err := os.Open(transferred)
	if err != nil {
		err = fmt.Errorf("transferred file: %w", err)
		return
	}
	defer t.Close()

	changed, diff, err = bindiff(f, t)
	return
}

func bindiff(f1, f2 *os.File) (changed bool, diff string, err error) {

	changed, diff = sizeDiff(f1, f2)
	if changed {
		return changed, diff, nil
	}

	return contentDiff(f1, f2)
}

func sizeDiff(f1, f2 *os.File) (changed bool, diff string) {
	st1, err := f1.Stat()
	if err != nil {
		err = fmt.Errorf("1st file stat: %w", err)
		return
	}
	st2, err := f2.Stat()
	if err != nil {
		err = fmt.Errorf("2nd file stat: %w", err)
		return
	}

	if sz1, sz2 := st1.Size(), st2.Size(); sz1 != sz2 {
		changed = true
		nameSize := max(len(f1.Name()), len(f2.Name()))

		diff = fmt.Sprintf(
			""+
				"--- %-[1]*[2]s  size: %[3]d\n"+
				"+++ %-[1]*[4]s  size: %[5]d\n",
			nameSize, f1.Name(), sz1, f2.Name(), sz2,
		)
	}
	return changed, diff
}

func contentDiff(f1, f2 *os.File) (changed bool, diff string, err error) {
	h1 := fnv.New64a()
	h2 := fnv.New64a()

	errc := make(chan error)

	// Note: runing both hashes page by page and comparing results early would finish faster
	// when the difference is early in the file, but then the sum diagnostics is lost
	go func() {
		_, err := io.Copy(h2, f2)
		errc <- err
	}()
	_, err = io.Copy(h1, f1)

	if err != nil {
		err = fmt.Errorf("1st file hash: %w", err)
		return
	}
	if err = <-errc; err != nil {
		err = fmt.Errorf("2nd file hash: %w", err)
		return
	}

	changed = h1.Sum64() != h2.Sum64()
	if changed {
		nameSize := max(len(f1.Name()), len(f2.Name()))

		diff = fmt.Sprintf(
			""+
				"--- %-[1]*[2]s  hash: %[3]s\n"+
				"+++ %-[1]*[4]s  hash: %[5]s\n",
			nameSize, f1.Name(), hexHash(h1), f2.Name(), hexHash(h2),
		)
	}
	return changed, diff, nil
}

func modeDiff(m1, m2 fs.FileMode) (changed bool, diff string) {
	if m1.IsRegular() && m2.IsRegular() && m2 != 000 && m1 != m2 {
		changed = true

		diff = fmt.Sprintf(
			""+
				"--- mode: 0%03[1]o\n"+
				"+++ mode: 0%03[2]o\n",
			m1, m2,
		)
	}
	return changed, diff
}

func hexHash(h hash.Hash) string {
	return hex.EncodeToString(h.Sum(nil))
}

func parseRsyncStats(s string) (stats string, err error) {
	var buf1, buf2 string
	var sent, recv, total int64
	var speedup float64

	if i := strings.Index(s, "sent "); i >= 0 {
		s = s[i:]
	} else {
		return "[no stats]", nil
	}
	_, err = fmt.Sscanf(s, "sent %s bytes received %s bytes", &buf1, &buf2)
	if err != nil {
		return "[invalid stats]", nil
	}
	r := strings.NewReplacer(",", "")
	sent, err = strconv.ParseInt(r.Replace(buf1), 10, 64)
	if err != nil {
		return "[invalid stats]", nil
	}
	recv, err = strconv.ParseInt(r.Replace(buf2), 10, 64)
	if err != nil {
		return "[invalid stats]", nil
	}
	stats = fmt.Sprintf("sent: %sB, recv: %sB", sizeFmt(sent), sizeFmt(recv))

	if i := strings.Index(s, "total size is "); i >= 0 {
		s = s[i:]
	} else {
		return stats, nil
	}
	_, err = fmt.Sscanf(s, "total size is %s  speedup is %s\n", &buf1, &buf2)
	if err != nil {
		return stats, nil
	}
	total, err = strconv.ParseInt(r.Replace(buf1), 10, 64)
	if err != nil {
		return stats, nil
	}
	stats += fmt.Sprintf(", total: %sB", sizeFmt(total))
	speedup, err = strconv.ParseFloat(r.Replace(buf2), 64)
	if err != nil {
		return stats, nil
	}
	stats += fmt.Sprintf(", speedup: %.2f", speedup)
	return stats, nil
}

func sizeFmt(size int64) string {
	units := []string{"", "Ki", "Mi", "Gi"}
	var x = float64(size)
	var j int
	for {
		y := x / 1024
		if math.Abs(y) < 1 || j >= len(units) {
			break
		}
		x = y
		j++
	}
	return fmt.Sprintf("%.3f%s", x, units[j])
}
