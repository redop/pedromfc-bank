// PedroBank server

package main

import (
	"flag"
	"fmt"
	"net/http"
	"strings"
)

// Starts the http server. The -certs argument for the binary indicates the
// location of the self-signed certificate and key.
func main() {
	var certs_dir string
	flag.StringVar(&certs_dir, "certs", ".",
		"Directory with key.pem and cert.pem")

	flag.Parse()

	var err = http.ListenAndServeTLS(
		"localhost:8080",
		strings.Join([]string{certs_dir, "/cert.pem"}, ""),
		strings.Join([]string{certs_dir, "/key.pem"}, ""),
		nil)

	if err != nil {
		fmt.Println(err)
	}
}
