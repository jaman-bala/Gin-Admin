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
	ErrInvalidPatchField  = errors.New("invalid patch field")
	ErrTooManyAttempts    = errors.New("too many failed login attempts for this account, try again later")
	ErrForbiddenRole      = errors.New("insufficient privileges for this operation")
	ErrLastSuperuser      = errors.New("cannot remove, deactivate or demote the last active superuser")
	ErrStaleWrite         = errors.New("resource was modified by another request, please reload and retry")
)
