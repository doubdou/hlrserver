package main

import (
	log "github.com/sirupsen/logrus"
)

func main() {

	// ams.Info.Println("this is info log")
	log.WithFields(log.Fields{
		"animal": "walrus",
	}).Info("A walrus appears")
}
