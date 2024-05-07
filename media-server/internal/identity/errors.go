package identity

import "errors"

var (
	ErrEmptyField                    = errors.New("empty field")
	ErrInvalidPassword               = errors.New("invalid password")
	ErrUserAlreadyExist              = errors.New("user already exist")
	ErrSameUserSignedByOnePrivateKey = errors.New("same user signed by one private key too much")
	ErrPrivateKeyNotFound            = errors.New("private key not found")
    ErrRefreshTokenConstraintViolation = errors.New("require refresh token")
)
