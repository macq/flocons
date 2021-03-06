package error

import (
	"fmt"
	"os"
	"syscall"
)

func NewIsDirError(path string) error {
	return &os.PathError{Op: "stat", Path: path, Err: syscall.EISDIR}
}

func NewIsNotDirError(path string) error {
	return &os.PathError{Op: "stat", Path: path, Err: syscall.ENOTDIR}
}

func IsIsDirError(err error) bool {
	pathError, ok := err.(*os.PathError)
	return ok && pathError.Err == syscall.EISDIR
}

func IsIsNotDirError(err error) bool {
	pathError, ok := err.(*os.PathError)
	return ok && pathError.Err == syscall.ENOTDIR
}

type ConfigError struct {
	Message string
}

func (e *ConfigError) Error() string {
	return fmt.Sprintf("Config error %s", e.Message)
}

func NewConfigError(message string) error {
	return &ConfigError{Message: message}
}

type HttpError struct {
	Status     string
	StatusCode int
}

func (e *HttpError) Error() string {
	return fmt.Sprintf("Http error %d: %s", e.StatusCode, e.Status)
}

func NewHttpError(status string, code int) error {
	return &HttpError{Status: status, StatusCode: code}
}

type InternalError struct {
	Reason string
}

func (e *InternalError) Error() string {
	return fmt.Sprintf("Internal error %s", e.Reason)
}

func NewInternalError(reason string) error {
	return &InternalError{Reason: reason}
}
