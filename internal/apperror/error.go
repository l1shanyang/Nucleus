package apperror

// Kind describes the stable business meaning of an application error.
type Kind string

const (
	KindValidation   Kind = "validation"
	KindNotFound     Kind = "not_found"
	KindConflict     Kind = "conflict"
	KindUnauthorized Kind = "unauthorized"
	KindForbidden    Kind = "forbidden"
	KindInternal     Kind = "internal"
)

// Error is a protocol-agnostic application error.
type Error struct {
	Kind    Kind
	Code    string
	Message string
	Err     error
}

func (e *Error) Error() string {
	if e.Err != nil {
		return e.Message + ": " + e.Err.Error()
	}
	return e.Message
}

func (e *Error) Unwrap() error {
	return e.Err
}

func newError(kind Kind, code, message string, err error) *Error {
	return &Error{
		Kind:    kind,
		Code:    code,
		Message: message,
		Err:     err,
	}
}

func Validation(message string) *Error {
	return newError(KindValidation, "BAD_REQUEST", message, nil)
}

func NotFound(message string) *Error {
	return newError(KindNotFound, "NOT_FOUND", message, nil)
}

func Conflict(message string) *Error {
	return newError(KindConflict, "CONFLICT", message, nil)
}

func Unauthorized(message string) *Error {
	return newError(KindUnauthorized, "UNAUTHORIZED", message, nil)
}

func Forbidden(message string) *Error {
	return newError(KindForbidden, "FORBIDDEN", message, nil)
}

func Internal(message string, err error) *Error {
	return newError(KindInternal, "INTERNAL", message, err)
}
