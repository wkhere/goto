package core

import (
	"strings"
	"text/template"
)

type Vars map[string]any

type Task interface {
	Desc() string
	View(Vars) (changed bool, diff string, err error)
	Exec(Vars) (changed bool, diff string, err error)
}

func Eval(templateName, input string, vars Vars) (string, error) {
	t, err := template.New(templateName).Parse(input)
	if err != nil {
		return "", err
	}
	b := new(strings.Builder)
	err = t.Execute(b, vars)
	return b.String(), err
}
