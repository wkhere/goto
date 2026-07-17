package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
)

type action struct {
	host     string
	tagsFlag string
	dryrun   bool

	help func()
}

func usage() {
	fmt.Println(`usage: goto HOST [-t,--tags=TAGS] [-n,--dry-run]`)
}

func parseArgs(args []string) (a action, _ error) {

	var p = PState{Args: args}

	for ; len(p.Args) > 0 && p.Err == nil; p.Next() {
		switch {

		case p.MatchJustFlag("-n", "--dry-run"):
			a.dryrun = true

		case p.MatchFlagExpr("-t", "--tags"):
			a.tagsFlag = "-t=" + p.Val

		case p.MatchJustFlag("-h", "--help"):
			a.help = usage
			return a, nil

		case p.MatchAnyFlag():
			p.Errorf("invalid flag: %s", p.Flag)

		default:
			if a.host != "" {
				p.Errorf("expected just 1 HOST arg")
			}
			a.host = p.Arg()
		}
	}

	switch {
	case p.Err != nil:
		return a, p.Err

	case a.host == "":
		return a, errors.New("no HOST given")
	}
	return a, nil
}

func main() {
	a, err := parseArgs(os.Args[1:])
	if err != nil {
		die(2, err)
	}
	if a.help != nil {
		a.help()
		os.Exit(0)
	}

	err = run(&a)
	if err != nil {
		die(1, err)
	}
}

func die(exitcode int, err error) {
	var errSubcmd *exec.ExitError
	if errors.As(err, &errSubcmd) {
		os.Exit(errSubcmd.ExitCode())
	}
	fmt.Fprintln(os.Stderr, "goto:", err)
	os.Exit(exitcode)
}
