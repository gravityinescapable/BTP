package main

import (
	"log"
	"net/http"

	"github.com/gravityinescapable/BTP/application/api/routes"
	"github.com/gravityinescapable/BTP/application/config"

	"github.com/gorilla/mux"
)

func main() {
	// Load the configuration
	err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Error loading config: %v", err)
	}

	// Create a new router
	r := mux.NewRouter()

	// Register routes
	routes.RegisterInvoiceRoutes(r)

	// Start the server
	port := config.GetConfig().Server.Port
	log.Printf("Starting server on port %s...", port)
	http.ListenAndServe(":"+port, r)
}
