package server

import (
	"fmt"
	"net/http"
)

var jsonErrFmt = `{"error": "%s"}` + "\n"

// An error that contains a JSON message safe to show to the client.
type publicJSONError struct {
	ErrJSON string
	ErrMsg  string // useful for testing
	Status  int
}

func (err *publicJSONError) Error() string {
	return err.ErrMsg
}

// Buold a public error. errMsg should be a simple error message, not in JSON,
// this function handles creating the JSON for it.
func newPublicError(status int, errMsg string) *publicJSONError {
	return &publicJSONError{
		fmt.Sprintf(jsonErrFmt, errMsg),
		errMsg,
		status}
}

// Generic errors
var invalidURLError = newPublicError(http.StatusNotFound, "invalid url")
var invalidMethodError = newPublicError(http.StatusMethodNotAllowed,
	"bad method for url")
var cantParseJSONError = newPublicError(http.StatusBadRequest,
	"can't parse request JSON")
var requestTooLongError = newPublicError(http.StatusRequestEntityTooLarge,
	"request too long")
var emptyRequestError = newPublicError(http.StatusBadRequest, "empty request")

// Account errors
var nameTooLongError = newPublicError(http.StatusBadRequest, "name too long")
var pwTooLongError = newPublicError(http.StatusBadRequest, "password too long")
var cpfInvalidError = newPublicError(http.StatusBadRequest, "bad CPF format")
var accExistsError = newPublicError(http.StatusBadRequest,
	"account already exists")
var idTooLargeError = newPublicError(http.StatusBadRequest,
	"id too large")
var noAccountError = newPublicError(http.StatusNotFound,
	"account does not exist")

// Login errors
var wrongPasswordError = newPublicError(http.StatusBadRequest,
	"wrong password")
var tryAgainError = newPublicError(http.StatusConflict,
	"please try again")
var unauthorizedError = newPublicError(http.StatusUnauthorized,
	"unauthorized")
var noTokenError = newPublicError(http.StatusBadRequest,
	"missing token")

// Transfer errors
var invalidAmountError = newPublicError(http.StatusBadRequest,
	"invalid amount")
var amountTooLargeError = newPublicError(http.StatusBadRequest,
	"amount too large")
