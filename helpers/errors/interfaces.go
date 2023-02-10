package errors

type ControllerError interface {
	error
	IsFinal() bool
}

func IsControllerError(err error) bool {
	_, ok := err.(ControllerError)
	return ok
}
