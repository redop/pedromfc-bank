package main

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
)

var logger = log.New(os.Stdout, "server: ", log.LstdFlags|log.Lmsgprefix)

// Respond to the client with an error. If err has a public error in its
// unrwap chain, respond with the message in that error. Otherwise,
// respond with a generic internal error message and log the error.
func respondWithError(rw http.ResponseWriter, err error) error {

	rw.Header().Set("Content-Type", "application/json;charset=UTF-8")

	var publicError *PublicJSONError
	var status int
	var errMsg string

	if errors.As(err, &publicError) {
		status = publicError.Status
		errMsg = publicError.Error()
	} else {
		status = http.StatusInternalServerError
		errMsg = fmt.Sprintf(jsonErrFmt, "internal server error")
		logger.Println(err)
	}

	rw.WriteHeader(status)

	_, writeErr := fmt.Fprint(rw, errMsg)

	if writeErr != nil {
		logger.Printf("Could not write response: %v", writeErr)
	}

	return writeErr
}
