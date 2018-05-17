package task

type Task interface {
	Query() (changed bool, diff string, err error)
	Apply() (changed bool, diff string, err error)
}
