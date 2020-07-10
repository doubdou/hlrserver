package main

import (
	"fmt"
	"hlr"

	_ "github.com/lib/pq"
)

func main() {
	hlr.LoadConfig()
	dbConf := hlr.DatabaseConfigGet()

	dbConn, err := hlr.OpenDBConnector(dbConf.Host, dbConf.Port, dbConf.User, dbConf.Password, dbConf.Name)
	if err != nil {
		fmt.Println("db connect", err)
		return
	}

	dbConn.CreateTable()

	httpConf := hlr.HTTPConfigGet()
	httpAddr := fmt.Sprintf("0.0.0.0:%s", httpConf.Port)
	if httpAddr == "" {
		fmt.Println("http addr config not valid.")
	}
	srv := hlr.NewWebServer()
	srv.Serve(httpAddr)
}
