package service

import "net/http"

type ErrorCode string

const (
	ErrCodeTeamExists  ErrorCode = "TEAM_EXISTS"
	ErrCodePRExists    ErrorCode = "PR_EXISTS"
	ErrCodePRMerged    ErrorCode = "PR_MERGED"
	ErrCodeNotAssigned ErrorCode = "NOT_ASSIGNED"
	ErrCodeNoCandidate ErrorCode = "NO_CANDIDATE"
	ErrCodeNotFound    ErrorCode = "NOT_FOUND"
)

type AppError struct {
	Code       ErrorCode
	HTTPStatus int
	Message    string
}

func (e *AppError) Error() string {
	return e.Message
}

func notFound(msg string) *AppError {
	return &AppError{
		Code:       ErrCodeNotFound,
		HTTPStatus: http.StatusNotFound,
		Message:    msg,
	}
}
