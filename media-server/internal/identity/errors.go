package identity

import "errors"

var (
	ErrEmptyField       = errors.New("empty field")
	ErrInvalidPassword  = errors.New("invalid password")
	ErrUserAlreadyExist = errors.New("user already exist")
)
