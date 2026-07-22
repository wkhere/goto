package core

import (
	"maps"
	"regexp"
	"strconv"
	"testing"
)

type testResult interface{ isTestResult() }
type errResult struct{ *regexp.Regexp }

type V = Vars
type S string

func (V) isTestResult()         {}
func (S) isTestResult()         {}
func (errResult) isTestResult() {}

func e(s string) errResult { return errResult{regexp.MustCompile(s)} }

func TestEval(t *testing.T) {
	tab := []struct {
		vars, subvars Vars
		want          testResult
	}{
		{V{}, nil, V{}},
		{V{}, V{}, V{}},
		{V{"Foo": 42}, nil, V{"Foo": 42}},
		{V{"Foo": 42.51}, nil, V{"Foo": 42.51}},
		{V{"Foo": "abc"}, nil, V{"Foo": "abc"}},

		{V{"Foo": "{{.X}}"}, nil, e("undefined var X used by Foo")},
		{V{"Foo": "{{.Foo}}"}, nil, e("cycle with Foo")},
		{V{"Foo": "{{print .Foo}}"}, nil, e("cycle with Foo")},
		{V{"X": 42, "Y": 5}, nil, V{"X": 42, "Y": 5}},
		{V{"X": "{{.Y}}", "Y": 42}, nil, V{"Y": 42, "X": "42"}},

		{V{"X": "{{print .Y}}", "Y": 42}, nil, V{"Y": 42, "X": "42"}},
		{V{"X": "{{.Y}}", "Y": "{{.Q}}"}, nil, e("undefined var Q used by Y")},
		{V{"X": "{{.Y}}", "Y": "{{.X}}"}, nil, e("cycle with (X|Y)")},
		{V{"X": "{{.Y}}", "Y": "{{.Q}}", "Q": 42}, nil, V{"Q": 42, "Y": "42", "X": "42"}},
		{V{"X": "{{.Y}}", "Y": "{{.Q}}", "Q": "{{.X}}"}, nil, e("cycle with (X|Y|Q)")},

		{V{"X": "{{.Y}}_{{.Q}}", "Y": "{{.Q}}", "Q": 1}, nil, V{"Q": 1, "Y": "1", "X": "1_1"}},
		{V{"X": "{{.Y}}_{{.Q}}", "Q": "{{.Y}}", "Y": 1}, nil, V{"Y": 1, "Q": "1", "X": "1_1"}},
		{V{"X": "{{.Y}}", "Y": "{{.A}}{{.A}}", "A": 1}, nil, V{"A": 1, "Y": "11", "X": "11"}},
		{V{"X": "{{.Y}}", "A": "{{.Y}}"}, nil, e("undefined var Y used by (X|A)")},
		{V{"X": "{{.Y}}_{{.Q}}", "Q": "{{.Y}}"}, nil, e("undefined var Y used by (X|Q)")},

		{
			V{"X": 42}, V{"Y": "{{.X}}"},
			V{"X": 42, "Y": "42"},
		},
		{
			V{"X": 42, "Y": "{{.X}}"}, V{"Q": "{{.Y}}"},
			V{"X": 42, "Y": "42", "Q": "42"},
		},
		{
			V{"X": 42, "Y": "{{.X}}"}, V{"Q": "{{.Foo}}"}, e("undefined var Foo used by Q"),
		},
		{
			V{"X": 42, "Y": "{{print .X}}"}, V{"Q": "{{print .Y}}"},
			V{"X": 42, "Y": "42", "Q": "42"},
		},
		{
			V{"X": 42, "Y": "{{print .X}}"}, V{"Q": `{{printf "%d_%s" .X .Y}}`},
			V{"X": 42, "Y": "42", "Q": "42_42"},
		},

		{V{"X": "{{"}, nil, e(`template: X:1: unclosed action`)},
		{V{"X": "{{.Y}}", "Y": "{{"}, nil, e(`template: Y:1: unclosed action`)},
		{V{}, V{"X": "{{"}, e(`template: X:1: unclosed action`)},
		{V{"X": "1"}, V{"X": "{{"}, e(`template: X:1: unclosed action`)},
	}

	for i, tc := range tab {
		t.Run(strconv.Itoa(i), func(t *testing.T) {

			rvars, err := tc.vars.Eval(Funcs, tc.subvars)

			switch errp, wantErr := tc.want.(errResult); {
			case !wantErr && err != nil:
				t.Errorf("unexpected error: %v", err)

			case wantErr && err == nil:
				t.Errorf("expected error with %q", errp)

			case wantErr && err != nil:
				if !errp.MatchString(err.Error()) {
					t.Errorf("expected error with %q; have: %q", errp, err)
				}

			default:
				evars := tc.want.(Vars)
				if !maps.Equal(rvars, evars) {
					t.Errorf("mismatch\nhave %#v\nwant %#v", rvars, evars)
				}
			}
		})
	}
}

func TestEvalWithExpr(t *testing.T) {
	tab := []struct {
		vars, subvars Vars
		expr          string
		want          testResult
	}{
		{V{}, nil, "foo", S("foo")},
		{V{}, nil, "{{.Foo}}", e(`undefined var Foo used by \$expr`)},
		{V{"Foo": 42}, nil, "{{.Foo}}", S("42")},
		{V{"Foo": 42.51}, nil, "{{.Foo}}", S("42.51")},
		{V{"Foo": "abc"}, nil, "{{.Foo}}", S("abc")},

		{V{"Foo": "{{.X}}"}, nil, "<any>", e("undefined var X used by Foo")},
		{V{"Foo": "{{.Foo}}"}, nil, "<any>", e("cycle with Foo")},
		{V{"Foo": "{{print .Foo}}"}, nil, "", e("cycle with Foo")},
		{V{"X": 42, "Y": 5}, nil, "{{.X}}_{{.Y}}", S("42_5")},
		{V{"X": "{{.Y}}", "Y": 1}, nil, "{{.X}}", S("1")},

		{V{"X": 42}, V{"Y": "{{.X}}"}, "{{.Y}}", S("42")},
		{V{"X": 42}, V{"Y": "x{{.X}}"}, "{{.X}}_{{.Y}}", S("42_x42")},
		{V{"X": 42}, V{"Y": "x{{.X}}"}, "{{.Q}}", e(`undefined var Q used by \$expr`)},
		{V{"X": 42, "Y": 5}, V{"Q": "{{.X}}_{{.Y}}"}, "{{.Q}}", S("42_5")},
		{V{"X": 42, "Y": 5}, V{"Q": "{{.Q}}"}, "<any>", e("cycle with Q")},

		{V{"X": "{{"}, nil, "<any>", e(`template: X:1: unclosed action`)},
		{V{"X": "{{.Y}}", "Y": "{{"}, nil, "<any>", e(`template: Y:1: unclosed action`)},
		{V{}, V{"X": "{{"}, "<any>", e(`template: X:1: unclosed action`)},
		{V{"X": "1"}, V{"X": "{{"}, "<any>", e(`template: X:1: unclosed action`)},
		{V{}, nil, "{{", e(`template: \$expr:1: unclosed action`)},
	}

	for i, tc := range tab {
		t.Run(strconv.Itoa(i), func(t *testing.T) {

			_, e, err := tc.vars.EvalWithExpr(tc.expr, Funcs, tc.subvars)

			switch errp, wantErr := tc.want.(errResult); {
			case !wantErr && err != nil:
				t.Errorf("unexpected error: %v", err)

			case wantErr && err == nil:
				t.Errorf("expected error with %q", errp)

			case wantErr && err != nil:
				if !errp.MatchString(err.Error()) {
					t.Errorf("expected error with %q; have: %q", errp, err)
				}

			default:
				v := string(tc.want.(S))
				if e != v {
					t.Errorf("mismatch\nhave %#v\nwant %#v", e, v)
				}
			}
		})
	}
}

func TestEvalExpr(t *testing.T) {
	tab := []struct {
		vars Vars
		expr string
		want testResult
	}{
		{V{}, "foo", S("foo")},
		{V{}, "{{.Foo}}", e(`undefined var Foo used by \$expr`)},
		{V{"Foo": 42}, "{{.Foo}}", S("42")},
		{V{"Foo": 42.51}, "{{.Foo}}", S("42.51")},
		{V{"Foo": "abc"}, "{{.Foo}}", S("abc")},

		{V{"Foo": "{{.X}}"}, "<any>", e("undefined var X used by Foo")},
		{V{"Foo": "{{.Foo}}"}, "<any>", e("cycle with Foo")},
		{V{"Foo": "{{print .Foo}}"}, "", e("cycle with Foo")},
		{V{"X": 42, "Y": 5}, "{{.X}}_{{.Y}}", S("42_5")},
		{V{"X": "{{.Y}}", "Y": 1}, "{{.X}}", S("{{.Y}}")},
		// NOTE for the above: contrary to EvalWithExpr, vars should be already evaluated

		{V{"X": "{{"}, "<any>", e(`template: X:1: unclosed action`)},
		{V{"X": "{{.Y}}", "Y": "{{"}, "<any>", e(`template: Y:1: unclosed action`)},
		{V{}, "{{", e(`template: \$expr:1: unclosed action`)},
	}

	for i, tc := range tab {
		t.Run(strconv.Itoa(i), func(t *testing.T) {

			e, err := tc.vars.EvalExpr(tc.expr, Funcs)

			switch errp, wantErr := tc.want.(errResult); {
			case !wantErr && err != nil:
				t.Errorf("unexpected error: %v", err)

			case wantErr && err == nil:
				t.Errorf("expected error with %q", errp)

			case wantErr && err != nil:
				if !errp.MatchString(err.Error()) {
					t.Errorf("expected error with %q; have: %q", errp, err)
				}

			default:
				v := string(tc.want.(S))
				if e != v {
					t.Errorf("mismatch\nhave %#v\nwant %#v", e, v)
				}
			}
		})
	}
}
