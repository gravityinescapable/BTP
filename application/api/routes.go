package main

import (
	"net/http"

	"github.com/gorilla/mux"
)

func RegisterRoutes() *mux.Router {
	r := mux.NewRouter()
	r.HandleFunc("/api/invoices", GetInvoices).Methods("GET")
	return r
}

func GetInvoices(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("Here is the list of invoices"))
}
