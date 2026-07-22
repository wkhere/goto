package core

import (
	"fmt"
	"maps"
	"slices"
	"strings"
	"text/template"
	"text/template/parse"
)

func (vars0 Vars) Eval(fmap FuncMap, subvars ...Vars) (rvars Vars, err error) {

	rvars, err = vars0.eval(fmap)
	if err != nil {
		return rvars, err
	}

	for _, lvars := range subvars {
		if Options.PrintVars {
			fmt.Printf("*** lvars: %#v\n", lvars)
		}
		rvars.merge(lvars)

		rvars, err = rvars.eval(fmap)
		if err != nil {
			return rvars, err
		}
	}

	if Options.PrintVars {
		fmt.Printf("*** rvars: %#v\n", rvars)
	}
	return rvars, nil
}

func (vars0 Vars) eval(fmap FuncMap) (rvars Vars, err error) {

	rvars = maps.Clone(vars0)

	env := makeVarEnv(len(vars0))

	err = env.feed(fmap, rvars)
	if err != nil {
		return rvars, err
	}

	order, err := env.resolveEvalOrder()
	if err != nil {
		return rvars, err
	}

	err = env.finishEvals(rvars, order)
	return rvars, err
}

func (vars0 Vars) merge(vars1 Vars) { maps.Insert(vars0, maps.All(vars1)) }

type varEnv struct {
	refs  map[string][]string
	evals map[string]*template.Template
}

func makeVarEnv(nvars int) (env varEnv) {
	env.refs = make(map[string][]string, nvars)
	env.evals = make(map[string]*template.Template, nvars)
	return
}

func (env varEnv) feed(fmap FuncMap, vars Vars) error {
	for name, value := range vars {
		env.markVar(name)
		if expr, ok := value.(string); ok {
			err := env.parseExpr(name, expr, fmap)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (env varEnv) resolveEvalOrder() (order []string, _ error) {

	tovisit := make(map[string]struct{}, len(env.refs))
	deps := make(map[string][]string, len(env.refs))

	for name, reflist := range env.refs {
		tovisit[name] = struct{}{}
		for _, ref := range reflist {
			if _, ok := env.refs[ref]; !ok {
				return nil, fmt.Errorf("undefined var %s used by %s", ref, name)
			}
			deps[ref] = append(deps[ref], name)
		}
	}

	tmark := make(map[string]struct{})
	var visit func(string) error

	visit = func(node string) error {
		if _, ok := tovisit[node]; !ok {
			return nil
		}
		if _, ok := tmark[node]; ok {
			return fmt.Errorf("cycle with %v", node)
		}

		tmark[node] = struct{}{}

		for _, m := range deps[node] {
			err := visit(m)
			if err != nil {
				return err
			}
		}

		delete(tovisit, node)
		order = append(order, node)
		return nil
	}

	for len(tovisit) > 0 {
		node := anyFromSet(tovisit)
		err := visit(node)
		if err != nil {
			return nil, err
		}
	}

	slices.Reverse(order)
	return order, nil
}

func anyFromSet[K comparable](m map[K]struct{}) K {
	for k := range m {
		return k
	}
	var zero K
	return zero
}

func (env varEnv) finishEvals(vars Vars, order []string) error {
	for _, name := range order {
		if t, ok := env.evals[name]; ok {
			b := new(strings.Builder)
			err := t.Execute(b, vars)
			if err != nil {
				return err
			}
			vars[name] = b.String()
		}
	}
	return nil
}

func (env varEnv) markVar(name string) {
	if _, ok := env.refs[name]; !ok {
		env.refs[name] = nil
	}
}

func (env varEnv) parseExpr(name, expr string, fmap FuncMap) error {
	t, err := template.New(name).Funcs(fmap).Parse(expr)
	if err != nil {
		return err
	}
	env.evals[name] = t

	c := make(chan string)
	go func() {
		pushFieldRefs(c, t.Tree.Root.Nodes)
		close(c)
	}()
	for ref := range c {
		env.refs[name] = append(env.refs[name], ref)
	}

	return nil
}

func pushFieldRefs(c chan<- string, nodes []parse.Node) {
	for _, node := range nodes {
		switch v := node.(type) {

		case *parse.ActionNode:
			pushFieldRefs(c, []parse.Node{v.Pipe})

		case *parse.PipeNode:
			subnodes := make([]parse.Node, len(v.Cmds))
			for i, cmd := range v.Cmds {
				subnodes[i] = cmd
			}
			pushFieldRefs(c, subnodes)

		case *parse.CommandNode:
			pushFieldRefs(c, v.Args)

		case *parse.FieldNode:
			c <- strings.Join(v.Ident, ".")
		}
	}
}

func (vars Vars) EvalExpr(expr string, fmap FuncMap) (string, error) {

	env := makeVarEnv(len(vars))
	err := env.feed(fmap, vars)
	if err != nil {
		return "", err
	}
	err = env.parseExpr("$expr", expr, fmap)
	if err != nil {
		return "", err
	}
	_, err = env.resolveEvalOrder()
	if err != nil {
		return "", err
	}

	b := new(strings.Builder)
	err = env.evals["$expr"].Execute(b, vars)
	return b.String(), err
}

func (vars Vars) EvalWithExpr(expr string, fmap FuncMap, subvars ...Vars) (Vars, string, error) {
	vars, err := vars.Eval(fmap, subvars...)
	if err != nil {
		return vars, "", err
	}
	s, err := vars.EvalExpr(expr, fmap)
	return vars, s, err
}
