package main

import (
	"log"
	"net/http"

	"github.com/gorilla/mux"
)

func main() {
	router := mux.NewRouter()
	routes.RegisterInvoiceRoutes(router)

	log.Println("Starting API server on port 8080...")
	log.Fatal(http.ListenAndServe(":8080", router))
}
