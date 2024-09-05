package main

import (
	"github.com/gravityinescapable/BTP/application/client"

	"github.com/gorilla/mux"
)

func RegisterInvoiceRoutes(router *mux.Router) {
	router.HandleFunc("/api/invoice", client.CreateOrUpdateInvoice).Methods("POST")
	router.HandleFunc("/api/purchases/{itemID}", client.GetTotalPurchases).Methods("GET")
	router.HandleFunc("/api/sales/{itemID}", client.GetTotalSales).Methods("GET")
	router.HandleFunc("/api/indices/{storeID}", client.GetIndices).Methods("GET")
	router.HandleFunc("/api/invalidate/{itemID}", client.InvalidateTransaction).Methods("POST")
}
