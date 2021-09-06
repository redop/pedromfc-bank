// PedroBank server

package server

import (
	"context"
	"fmt"
	"net/http"
	"strings"
)

const welcomeSring = "Welcome to PedroBank!\n"

// Respond with a welcome message when client GETs path /
// Respond with an error for other paths matched by this function.
func welcomeResponse(rw http.ResponseWriter, req *http.Request) {
	if req.URL.Path != "/" {
		respondWithError(rw, invalidURLError)
		return
	}

	if req.Method != http.MethodGet {
		respondWithError(rw, invalidMethodError)
		return
	}

	setJSONEncoding(rw)
	_, err := fmt.Fprint(rw, welcomeSring)

	if err != nil {
		logger.Printf("could not write response: %v", err)
	}
}

func ping(rw http.ResponseWriter, req *http.Request) {
	if req.URL.Path != "/ping" {
		respondWithError(rw, invalidURLError)
		return
	}

	if req.Method != http.MethodGet {
		respondWithError(rw, invalidMethodError)
		return
	}

	rw.WriteHeader(http.StatusOK)
}

var server http.Server = http.Server{Addr: "localhost:8080"}

// Channel that signals when the server finishes
var ServerFinished = make(chan interface{})

// Channel that signals when the login cleaner finishes
var loginCleanerFinished = make(chan interface{})

// Starts the http server. The -certs argument for the binary indicates the
// location of the self-signed certificate and key.
func Run(certsDir string) {
	var err error

	http.HandleFunc("/ping", ping)
	http.HandleFunc("/", welcomeResponse)
	http.HandleFunc("/accounts", handleAccounts)
	http.HandleFunc("/accounts/", getAccountBalance)
	http.HandleFunc("/login", login)
	http.HandleFunc("/id", getId)
	http.HandleFunc("/transfers", handleTransfers)

	loginCleanerContext, loginCleanerCancelFunc := context.WithCancel(
		context.Background())

	go loginClean(loginCleanerContext)

	logger.Println("Starting PedroBank server")

	err = server.ListenAndServeTLS(
		strings.Join([]string{certsDir, "/cert.pem"}, ""),
		strings.Join([]string{certsDir, "/key.pem"}, ""))

	logger.Print(err)

	loginCleanerCancelFunc()

	// Wait for the login cleaner to finish
	<-loginCleanerFinished

	close(ServerFinished)
}

func Stop() {
	server.Shutdown(context.Background())
}
