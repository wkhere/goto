package handler

import (
	"strings"
	"fmt"
	"os/exec"
)

type SystemctlRestart struct{ Service string }

func (sr SystemctlRestart) Run() error {
	cmd := exec.Command("sudo", "systemctl", "restart", sr.Service)
	err := cmd.Run()
	if err != nil {
		return err
	}
	st := exec.Command("sudo", "systemctl", "status", sr.Service)
	b := new(strings.Builder)
	st.Stdout = b
	err = st.Run()
	if err != nil {
		fmt.Printf("=== %#v dump:\n", sr)
		fmt.Println(b.String())
		return err
	}
	return nil
}
