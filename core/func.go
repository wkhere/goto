package core

import (
	"os"
	"text/template"
)

type FuncMap = template.FuncMap

var Funcs = FuncMap{
	"getenv": os.Getenv,
}
