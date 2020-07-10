package main

import (
	"hlr"

	log "github.com/sirupsen/logrus"
)

func main() {

	hlr.Info.Println("this is info log")
	log.WithFields(log.Fields{
		"animal": "walrus",
	}).Info("A walrus appears")
}
