package main

import (
	"fmt"
	"net/http"
)

var jsonErrFmt = `{"error": "%s"}` + "\n"

// An error that contains a JSON message safe to show to the client.
type PublicJSONError struct {
	ErrJSON string
	Status  int
}

func (err *PublicJSONError) Error() string {
	return err.ErrJSON
}

// Buold a public error. errMsg should be a simple error message, not in JSON,
// this function handles creating the JSON for it.
func NewPublicError(status int, errMsg string) *PublicJSONError {
	return &PublicJSONError{
		fmt.Sprintf(jsonErrFmt, errMsg),
		status}
}

// Generic errors
var InvalidURLError = NewPublicError(http.StatusNotFound, "invalid url")
var InvalidMethodError = NewPublicError(http.StatusMethodNotAllowed,
	"bad method for url")
