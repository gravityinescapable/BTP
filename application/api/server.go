package main

import (
	"log"
	"net/http"
)

func main() {
	r := RegisterRoutes()
	log.Fatal(http.ListenAndServe(":8080", r))
}
