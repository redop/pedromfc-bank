package server

import (
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"regexp"
)

var logger = log.New(os.Stdout, "server: ", log.LstdFlags|log.Lmsgprefix)

// Redefine in32 as money so that we can use make a MarshalJSON method for it.
// We represent the account balance as the actual balance time 100 (e.g. BRL
// 223.15 is represented as 22315). We only support addition/substraction so
// we don't need more decimals than two, for the BRL cents.
type money int32

// Stringify a money value. The last two digits are the cents.
func (num money) String() string {
	return fmt.Sprintf("%d.%d%d", num/100, (num%100)/10, (num%100)%10)
}

func (num money) MarshalJSON() ([]byte, error) {
	return []byte(num.String()), nil
}

// Respond to the client with an error. If err has a public error in its
// unrwap chain, respond with the message in that error. Otherwise,
// respond with a generic internal error message and log the error.
func respondWithError(rw http.ResponseWriter, err error) error {

	setJSONEncoding(rw)

	var publicError *publicJSONError
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

// Read from the request body up to maxLen bytes, and return in error if the
// request is too long or empty.
func readFromReq(req *http.Request, maxLen int) ([]byte, error) {
	if maxLen <= 0 {
		return nil, fmt.Errorf("invalid maxLen")
	}

	var out = make([]byte, 0, maxLen)
	var readBuf = make([]byte, maxLen)

	n, err := req.Body.Read(readBuf)

	for err == nil || (err == io.EOF && n > 0) {
		if len(out)+n > maxLen {
			return nil, requestTooLongError
		}

		// This should never re-allocate an underlying array since we made
		// the slice with maxLen as its capacity.
		out = append(out, readBuf[:n]...)

		if err == nil {
			n, err = req.Body.Read(readBuf)
		} else {
			// eof
			break
		}
	}

	if err == io.EOF {
		if len(out) == 0 {
			return nil, emptyRequestError
		} else {
			return out, nil
		}
	} else {
		return nil, err
	}
}

func setJSONEncoding(rw http.ResponseWriter) {
	rw.Header().Set("Content-Type", "application/json;charset=UTF-8")
}

var cpfRegex *regexp.Regexp = regexp.MustCompile(
	`^[0-9]{3}\.[0-9]{3}-[0-9]{2}$`)
