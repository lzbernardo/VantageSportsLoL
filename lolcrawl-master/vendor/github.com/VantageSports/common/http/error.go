package http

import "fmt"

type Error struct {
	IsErr bool   `json:"error"`
	Code  int    `json:"code"`
	Msg   string `json:"message"`
}

func NewError(code int, err error) *Error {
	return &Error{true, code, err.Error()}
}

func (e *Error) Error() string {
	return fmt.Sprintf("Code: %d, Message: %s", e.Code, e.Msg)
}
