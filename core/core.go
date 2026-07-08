package core

import (
	"slices"
	"strings"
	"text/template"
)

type Vars map[string]any

type Task interface {
	Desc() string
	View(Vars) (changed bool, diff string, notify []Handler, err error)
	Exec(Vars) (changed bool, diff string, notify []Handler, err error)
}

type Transfer interface {
	Desc() string
	RemotePrep(Vars) (stats string, err error)
	DoTransfer(Vars) (stats string, err error)
}

type Handler interface {
	Run() error
}

type Notifier []Handler

const (
	MetavarRemote = "$remote"
)

func Eval(templateName, input string, vars Vars) (string, error) {
	t, err := template.New(templateName).Parse(input)
	if err != nil {
		return "", err
	}
	b := new(strings.Builder)
	err = t.Execute(b, vars)
	return b.String(), err
}

func DedupHandlers(hh *[]Handler) {
	visited := make(map[Handler]struct{}, len(*hh))

	*hh = slices.DeleteFunc(*hh, func(h Handler) bool {
		if _, ok := visited[h]; ok {
			return true
		}
		visited[h] = struct{}{}
		return false
	})
}
