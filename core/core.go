package core

import (
	"slices"
)

type (
	Vars map[string]any

	Task interface {
		Desc() string
		View(Vars) (changed bool, diff string, notify []Handler, err error)
		Exec(Vars) (changed bool, diff string, notify []Handler, err error)
	}

	Transfer interface {
		Desc() string
		RemotePrep(Vars) (stats string, err error)
		DoTransfer(Vars) (stats string, err error)
	}

	Handler interface {
		Run() error
	}

	Notifier []Handler
)

const (
	MetavarRemote = "$remote"
)

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
