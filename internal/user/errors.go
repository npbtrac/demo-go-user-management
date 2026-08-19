package user

import "errors"

var (
	ErrNotFound = errors.New("user not found")
	ErrConflict = errors.New("user already exists")
	ErrInvalid  = errors.New("invalid user payload")
)
