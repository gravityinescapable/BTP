package main

import (
	"io"
	"log"
	"net/http"
)

func FetchInvoices() {
	resp, err := http.Get("http://localhost:8080/api/invoices")
	if err != nil {
		log.Fatal(err)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}
	log.Println(string(body))
}
