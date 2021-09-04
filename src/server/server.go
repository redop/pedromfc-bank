// PedroBank server

package server

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"strings"
)

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
	_, err := fmt.Fprintf(rw, "Welcome to PedroBank!\n")

	if err != nil {
		logger.Printf("could not write response: %v", err)
	}
}

var server http.Server = http.Server{Addr: "localhost:8080"}

// Channel that signals when the server finishes
var ServerFinished = make(chan interface{})

// Starts the http server. The -certs argument for the binary indicates the
// location of the self-signed certificate and key.
func Run() {
	var certs_dir string
	flag.StringVar(&certs_dir, "certs", ".",
		"Directory with key.pem and cert.pem")

	flag.Parse()

	var err = openDBPool()
	defer db.Close()

	if err != nil {
		logger.Fatal(err)
		return
	}

	http.HandleFunc("/", welcomeResponse)
	http.HandleFunc("/accounts", handleAccounts)
	http.HandleFunc("/accounts/", getAccountBalance)

	err = server.ListenAndServeTLS(
		strings.Join([]string{certs_dir, "/cert.pem"}, ""),
		strings.Join([]string{certs_dir, "/key.pem"}, ""))

	logger.Print(err)

	close(ServerFinished)
}

func Stop() {
	server.Shutdown(context.Background())
}
