package chainpot

type ErrorHandler func(err error)

var errorHandler ErrorHandler

func SetErrorHandler(fn ErrorHandler) {
	errorHandler = fn
}

func reportError(err error) {
	if errorHandler != nil {
		errorHandler(err)
	}
}
