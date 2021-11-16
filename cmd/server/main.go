package main

import (
	"github.com/sharop/lab_goconnect/internal/server"
	"log"
)

func main() {

	srv := server.NewHTTPServer(":9010")
	log.Fatal(srv.ListenAndServe())
}
