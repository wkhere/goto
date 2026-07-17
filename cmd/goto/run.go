package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	_ "embed"
)

const (
	agentDir     = "agent"
	targetLocal  = ".agent.local"
	targetRemote = ".agent.remote"
	targetShort  = ".agent"
)

//go:embed agent_main.go.src
var agentMain []byte

func injectMain() (err error) {
	f, err := os.Create(filepath.Join(agentDir, "main.go"))
	if err != nil {
		return err
	}
	defer safeClose(f, &err)

	_, err = f.Write(agentMain)
	return err
}

func build(gocmd, target, goos, goarch string) error {
	cmd := exec.Command(gocmd, "build", "-ldflags=-s", "-o", target, "./"+agentDir)

	if goos != "" {
		cmd.Env = append(cmd.Environ(), "GOOS="+goos)
	}
	if goarch != "" {
		cmd.Env = append(cmd.Environ(), "GOARCH="+goarch)
	}
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout

	return cmd.Run()
}

func transferAgent(host string) error {
	// TODO: suffixes for the transferred binary!
	cmd := exec.Command("rsync", "-c", targetRemote, host+(":"+targetShort))

	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout

	return cmd.Run()
}

func doLocal(host string, tagsConcat string, verb string) error {
	var cmd *exec.Cmd
	if tagsConcat == "" {
		cmd = exec.Command("./"+targetLocal, "-r", host, verb)
	} else {
		cmd = exec.Command("./"+targetLocal, "-t", tagsConcat, "-r", host, verb)
	}

	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout

	return cmd.Run()
}

func doRemote(host string, tagsConcat string, verb string) error {
	var cmd *exec.Cmd
	if tagsConcat == "" {
		cmd = exec.Command("ssh", host, "./"+targetShort, verb)
	} else {
		cmd = exec.Command("ssh", host, "./"+targetShort, "-t", tagsConcat, verb)
	}
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout

	return cmd.Run()
}

func run(a *action) error {
	// here goes the code that compiles, sends and runs the agent

	err := injectMain()
	if err != nil {
		return fmt.Errorf("inject main: %w", err)
	}

	gocmd, err := exec.LookPath("go")
	if err != nil {
		return err
	}

	fmt.Println("= build local agent")
	err = build(gocmd, targetLocal, "", "")
	if err != nil {
		return err
	}
	fmt.Println("= build remote agent")
	err = build(gocmd, targetRemote, os.Getenv("HOSTOS"), os.Getenv("HOSTARCH"))
	if err != nil {
		return err
	}
	fmt.Println("= transfer remote agent")
	err = transferAgent(a.host)
	if err != nil {
		return err
	}
	fmt.Println("")

	err = doRemote(a.host, a.tagsConcat, "prep")
	if err != nil {
		return err
	}
	err = doLocal(a.host, a.tagsConcat, "transfer")
	if err != nil {
		return err
	}
	var actverb = "exec"
	if a.dryrun {
		actverb = "view"
	}
	err = doRemote(a.host, a.tagsConcat, actverb)
	return err
}

func safeClose(f *os.File, err *error) {
	if errc := f.Close(); errc != nil && *err == nil {
		*err = errc

	}
}
