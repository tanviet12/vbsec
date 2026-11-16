package service

import "errors"

var (
	ErrBadCredentials = errors.New("invalid credentials")
	ErrBadTicket      = errors.New("invalid or expired ticket")
	ErrBadCode        = errors.New("invalid code")
	ErrLocked         = errors.New("too many attempts")
	ErrForbidden      = errors.New("forbidden")
	ErrNotFound       = errors.New("not found")
	ErrConflict       = errors.New("conflict")
	ErrInvalid        = errors.New("invalid input")
	ErrPeriodClosed   = errors.New("period is closed")
	ErrOpenRequests   = errors.New("period has open requests")
	ErrOverLimit      = errors.New("department limit exceeded")
	ErrInsufficient   = errors.New("insufficient fund balance")
)
