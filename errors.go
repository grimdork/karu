package karu

import (
	"errors"
	"fmt"
)

var (
	ErrPostNotFound      = errors.New("post not found")
	ErrThreadNotFound    = errors.New("thread not found")
	ErrPermissionDenied  = errors.New("permission denied")
	ErrThreadLocked      = errors.New("thread is locked")
	ErrInvalidPermission = errors.New("invalid permission string")
	ErrPostHasReplies    = errors.New("post has replies and cannot be deleted")
	ErrInvalidPath       = errors.New("invalid path")
	ErrInvalidParent     = errors.New("parent post not found for move")
)

type PermissionError struct {
	Path string
	Code byte
}

func (e *PermissionError) Error() string {
	return fmt.Sprintf("permission denied: missing '%c' on path %q", e.Code, e.Path)
}

func (e *PermissionError) Unwrap() error {
	return ErrPermissionDenied
}
