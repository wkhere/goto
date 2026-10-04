package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	_ "embed"
)

const (
	actorDir     = "act"
	targetPrefix = ".actor"
)

type (
	cmd    string
	envv   string
	host   string
	target string
	verb   string
	tags   string
)

var envnil = envv("")

func getenv(key string) envv { return envv(os.Getenv(key)) }

func (t target) add(s string) target { return target(string(t) + s) }

//go:embed actor_main.go.src
var actorMain []byte

func injectMain() (err error) {
	f, err := os.Create(filepath.Join(actorDir, "main.go"))
	if err != nil {
		return err
	}
	defer safeClose(f, &err)

	_, err = f.Write(actorMain)
	return err
}

func build(gocmd cmd, goos, goarch envv, target target) error {
	cmd := exec.Command(string(gocmd), "build", "-ldflags=-s", "-o", string(target), "./"+actorDir)

	if goos != "" {
		cmd.Env = append(cmd.Environ(), "GOOS="+string(goos))
	}
	if goarch != "" {
		cmd.Env = append(cmd.Environ(), "GOARCH="+string(goarch))
	}
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout

	return cmd.Run()
}

func transferActor(targetSrc, targetDst target, host host) error {
	cmd := exec.Command("rsync", "-c", string(targetSrc), string(host)+(":"+string(targetDst)))

	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout

	return cmd.Run()
}

func doLocal(target target, verb verb, tagsConcat tags, host host, printVars bool) error {

	args := []string{string(verb), "-r", string(host)}

	if tagsConcat != "" {
		args = append(args, "-t", string(tagsConcat))
	}
	if printVars {
		args = append(args, "-v")
	}

	cmd := exec.Command("./"+string(target), args...)
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout

	return cmd.Run()
}

func doRemote(host host, target target, verb verb, tagsConcat tags, printVars bool) error {

	args := []string{string(host), "./" + string(target), string(verb)}

	if tagsConcat != "" {
		args = append(args, "-t", string(tagsConcat))
	}
	if printVars {
		args = append(args, "-v")
	}

	cmd := exec.Command("ssh", args...)
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout

	return cmd.Run()
}

func run(a *action) error {
	// here goes the code that compiles, sends and runs the actor

	err := injectMain()
	if err != nil {
		return fmt.Errorf("inject main: %w", err)
	}

	gocmdpath, err := exec.LookPath("go")
	if err != nil {
		return err
	}
	gocmd := cmd(gocmdpath)

	hash, err := hashFromLocalWD()
	if err != nil {
		return err
	}
	target := target(targetPrefix + "-" + hash)

	fmt.Println("= build local actor")
	err = build(gocmd, envnil, envnil, target.add(".local"))
	if err != nil {
		return err
	}
	fmt.Println("= build remote actor")
	err = build(gocmd, getenv("HOSTOS"), getenv("HOSTARCH"), target.add(".remote"))
	if err != nil {
		return err
	}
	fmt.Println("= transfer remote actor")
	err = transferActor(target.add(".remote"), target, a.host)
	if err != nil {
		return err
	}
	fmt.Println("")

	err = doRemote(a.host, target, "prep", a.tagsConcat, a.printVars)
	if err != nil {
		return err
	}
	err = doLocal(target.add(".local"), "transfer", a.tagsConcat, a.host, a.printVars)
	if err != nil {
		return err
	}
	var actverb verb = "exec"
	if a.dryrun {
		actverb = "view"
	}
	err = doRemote(a.host, target, actverb, a.tagsConcat, a.printVars)
	return err
}

func safeClose(f *os.File, err *error) {
	if errc := f.Close(); errc != nil && *err == nil {
		*err = errc

	}
}
