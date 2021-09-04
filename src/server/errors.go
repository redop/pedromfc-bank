package main

import (
	"fmt"
	"net/http"
)

var jsonErrFmt = `{"error": "%s"}` + "\n"

type PublicJSONError struct {
	ErrJSON string
	Status  int
}

func (err *PublicJSONError) Error() string {
	return err.ErrJSON
}

func NewPublicError(status int, errMsg string) *PublicJSONError {
	return &PublicJSONError{
		fmt.Sprintf(jsonErrFmt, errMsg),
		status}
}

// Generic errors
var InvalidURLError = NewPublicError(http.StatusNotFound, "invalid url")
var InvalidMethodError = NewPublicError(http.StatusMethodNotAllowed,
	"bad method for url")
