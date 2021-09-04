package main

import (
	"fmt"
	"net/http"
)

var jsonErrFmt = `{"error": "%s"}` + "\n"

// An error that contains a JSON message safe to show to the client.
type publicJSONError struct {
	ErrJSON string
	Status  int
}

func (err *publicJSONError) Error() string {
	return err.ErrJSON
}

// Buold a public error. errMsg should be a simple error message, not in JSON,
// this function handles creating the JSON for it.
func newPublicError(status int, errMsg string) *publicJSONError {
	return &publicJSONError{
		fmt.Sprintf(jsonErrFmt, errMsg),
		status}
}

// Generic errors
var invalidURLError = newPublicError(http.StatusNotFound, "invalid url")
var invalidMethodError = newPublicError(http.StatusMethodNotAllowed,
	"bad method for url")
