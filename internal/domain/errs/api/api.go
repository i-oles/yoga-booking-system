package api

const (
	BadRequestCode = iota
	ConflictCode
	NotFoundCode
)

type APIError struct {
	Code int
	Err  error
}

func (e APIError) Error() string {
	return e.Err.Error()
}

func ErrValidation(err error) *APIError {
	return &APIError{
		Code: BadRequestCode,
		Err:  err,
	}
}

func ErrClassNotEmpty(err error) *APIError {
	return &APIError{
		Code: ConflictCode,
		Err:  err,
	}
}

func ErrPreviousPassNotFinished(err error) *APIError {
	return &APIError{
		Code: ConflictCode,
		Err:  err,
	}
}

func ErrNotFound(err error) *APIError {
	return &APIError{
		Code: NotFoundCode,
		Err:  err,
	}
}
