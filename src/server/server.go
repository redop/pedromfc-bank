// PedroBank server

package main

import (
	"flag"
	"fmt"
	"net/http"
	"strings"
)

// Respond with a welcome message when client GETs path /
// Respond with an error for other paths matched by this function.
func welcomeResponse(rw http.ResponseWriter, req *http.Request) {
	if req.URL.Path != "/" {
		respondWithError(rw, InvalidURLError)
		return
	}

	if req.Method != http.MethodGet {
		respondWithError(rw, InvalidMethodError)
		return
	}

	rw.Header().Set("Content-Type", "application/json;charset=UTF-8")
	_, err := fmt.Fprintf(rw, "Welcome to PedroBank!\n")

	if err != nil {
		logger.Printf("could not write response: %v", err)
	}
}

// Starts the http server. The -certs argument for the binary indicates the
// location of the self-signed certificate and key.
func main() {
	var certs_dir string
	flag.StringVar(&certs_dir, "certs", ".",
		"Directory with key.pem and cert.pem")

	flag.Parse()

	http.HandleFunc("/", welcomeResponse)

	var err = http.ListenAndServeTLS(
		"localhost:8080",
		strings.Join([]string{certs_dir, "/cert.pem"}, ""),
		strings.Join([]string{certs_dir, "/key.pem"}, ""),
		nil)

	if err != nil {
		fmt.Println(err)
	}
}
