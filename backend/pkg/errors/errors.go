package errors

import "errors"

var (
	ErrUserNotFound       = errors.New("user not found")
	ErrInvalidUUID        = errors.New("invalid uuid")
	ErrAccountBlocked     = errors.New("account blocked")
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrInvalidToken       = errors.New("invalid token")
	ErrPhoneAlreadyExists = errors.New("phone already exists")
	ErrConflict           = errors.New("conflict")
)
