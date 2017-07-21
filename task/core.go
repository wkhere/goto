package task

type Task interface {
	Apply() (changed bool, err error)
}
