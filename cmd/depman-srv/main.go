package main

import (
	log "github.com/Sirupsen/logrus"
	"github.com/moensch/depman"
)

func main() {
	log.SetLevel(log.DebugLevel)

	srv, err := depman.NewServer()
	if err != nil {
		log.Fatalf("Cannot start server: %s", err)
	}
	log.Info("initialized")

	srv.Run()
}
