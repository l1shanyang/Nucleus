package service

// ValidationError 表示由用户输入导致的业务校验错误。
type ValidationError struct {
	Message string
}

func (e *ValidationError) Error() string {
	return e.Message
}

func validationError(message string) *ValidationError {
	return &ValidationError{Message: message}
}
