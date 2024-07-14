package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	this "github.com/74th/remote-command-http-server"
	"github.com/alecthomas/kingpin/v2"
)

var (
	configPath = kingpin.Arg("config", "Path to config file").Required().String()
	port       = kingpin.Flag("port", "Port number").Short('p').Default("8080").Int()
)

func main() {
	kingpin.Parse()
	sv, err := this.NewServer(fmt.Sprintf(":%d", *port), *configPath)

	if err != nil {
		log.Printf("failed to create server: %v", err)
		os.Exit(1)
	}

	if err := sv.Start(); err != nil {
		log.Printf("failed to start server: %v", err)
		os.Exit(1)
	}

	log.Printf("Server started")

	sigC := make(chan os.Signal, 1)
	signal.Notify(sigC, syscall.SIGINT, syscall.SIGTERM)
	<-sigC

	if err := sv.Shoutdown(); err != nil {
		log.Printf("failed to shoutdown server: %v", err)
		os.Exit(1)
	}
}
