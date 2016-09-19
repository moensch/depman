package main

import (
	"flag"
	log "github.com/Sirupsen/logrus"
	"github.com/moensch/depman"
)

var (
	logLevel   string
	listenAddr string
	storeDir   string
)

func init() {
	flag.StringVar(&logLevel, "d", "info", "Log level (debug|info|warn|error|fatal)")
	flag.StringVar(&listenAddr, "l", "0.0.0.0:8082", "Listen address and port")
	flag.StringVar(&storeDir, "s", "/tmp/depman_files", "Data storage directory")
}

func main() {
	flag.Parse()

	lvl, _ := log.ParseLevel(logLevel)
	log.SetLevel(lvl)

	srv, err := depman.NewServer()
	if err != nil {
		log.Fatalf("Cannot start server: %s", err)
	}
	depman.StoreDir = storeDir
	log.Info("initialized")

	srv.Run(listenAddr)
}
