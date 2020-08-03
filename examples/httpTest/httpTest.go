package main

import (
	"ams"
	"fmt"

	_ "github.com/lib/pq"
)

func main() {
	ams.LoadConfig()
	dbConf := ams.DatabaseConfigGet()

	dbConn, err := ams.OpenDBConnector(dbConf.Host, dbConf.Port, dbConf.User, dbConf.Password, dbConf.Name)
	if err != nil {
		fmt.Println("db connect", err)
		return
	}

	dbConn.CreateTable()

	httpConf := ams.HTTPConfigGet()
	httpAddr := fmt.Sprintf("0.0.0.0:%s", httpConf.Port)
	if httpAddr == "" {
		fmt.Println("http addr config not valid.")
	}
	srv := ams.NewWebServer()
	srv.Serve(httpAddr)
}
