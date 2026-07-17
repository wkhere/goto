package core

import (
	"maps"
	"slices"
	"strings"
	"text/template"
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

	FuncMap = template.FuncMap
)

const (
	MetavarRemote = "$remote"
)

func (vars0 Vars) Eval(fmap FuncMap, subvars ...Vars) (rvars Vars, err error) {
	// Note: vars from the same level can't reference each other.
	// So, first the receiver vars are evaluated for the presence of functions.
	// Then, each of levels of subvars is evaluated for the presence of both functions and
	// higher-level vars.
	// This is not ideal, but is the best approach atm.

	rvars = maps.Clone(vars0)

	for k, v := range rvars {
		if s, ok := v.(string); ok {
			e, err := evalVar(s, fmap, nil)
			if err != nil {
				return rvars, err
			}
			rvars[k] = e
		}
	}

	for _, lvars := range subvars {
		pvars := maps.Clone(rvars)

		for k, v := range lvars {
			if s, ok := v.(string); ok {
				e, err := evalVar(s, fmap, pvars)
				if err != nil {
					return rvars, err
				}
				rvars[k] = e
			}
		}
	}
	return rvars, nil
}

func evalVar(v string, fmap FuncMap, vars Vars) (string, error) {
	t, err := template.New("var").Funcs(fmap).Parse(v)
	if err != nil {
		return "", err
	}
	b := new(strings.Builder)
	err = t.Execute(b, vars)
	return b.String(), err
}

func (vars Vars) EvalExpr(expr string) (string, error) {
	t, err := template.New("expr").Parse(expr)
	if err != nil {
		return "", err
	}
	b := new(strings.Builder)
	err = t.Execute(b, vars)
	return b.String(), err
}

func (vars Vars) EvalWithExpr(expr string, fmap FuncMap, subvars ...Vars) (Vars, string, error) {
	vars, err := vars.Eval(fmap, subvars...)
	if err != nil {
		return vars, "", err
	}
	s, err := vars.EvalExpr(expr)
	return vars, s, err
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
