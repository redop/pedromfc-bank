package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"pedro-bank/server"
)

func main() {
	sigintStop := make(chan os.Signal, 1)
	signal.Notify(sigintStop, os.Interrupt)

	var certsDir string
	flag.StringVar(&certsDir, "certs", ".",
		"Directory with key.pem and cert.pem")

	flag.Parse()

	go server.Run(certsDir)

	<-sigintStop
	close(sigintStop)
	fmt.Println("Got sigint")

	server.Stop()

	<-server.ServerFinished
}
