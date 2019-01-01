package main

import (
	"log"
	"net/http"
)

func main() {
	// Simple static webserver:
	log.Println("http server listen on :1999")
	log.Fatal(http.ListenAndServe(":1999", http.FileServer(http.Dir("./"))))
}
