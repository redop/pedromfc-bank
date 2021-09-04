package main

import (
	"flag"
	"pedro-bank/server"
)

func main() {
	var certsDir string
	flag.StringVar(&certsDir, "certs", ".",
		"Directory with key.pem and cert.pem")

	flag.Parse()

	server.Run(certsDir)
}
