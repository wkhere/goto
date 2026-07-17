package core

import (
	"os"
)

var Funcs = FuncMap{
	"getenv": os.Getenv,
}
