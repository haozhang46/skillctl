package app

import "fmt"

const (
	ExitOK    = 0
	ExitFail  = 1
	ExitUsage = 2
)

// Error is a CLI-facing failure with a stable exit code.
type Error struct {
	ExitCode int
	Kind     string
	Message  string
	Path     string
	Hint     string
}

func (e *Error) Error() string {
	if e.Path != "" {
		return fmt.Sprintf("%s: %s", e.Path, e.Message)
	}
	return e.Message
}

func Fail(kind, message string) *Error {
	return &Error{ExitCode: ExitFail, Kind: kind, Message: message}
}

func Usage(message string) *Error {
	return &Error{ExitCode: ExitUsage, Kind: "usage", Message: message}
}

func WithPath(err *Error, path string) *Error {
	err.Path = path
	return err
}

func WithHint(err *Error, hint string) *Error {
	err.Hint = hint
	return err
}

func AsError(err error) *Error {
	if err == nil {
		return nil
	}
	if e, ok := err.(*Error); ok {
		return e
	}
	return Fail("internal", err.Error())
}
