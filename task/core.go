package task

type Task interface {
	Apply(options interface{}) (changed bool, err error)
}
