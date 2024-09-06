package client

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
)

func CreateOrUpdateInvoice(w http.ResponseWriter, r *http.Request) {
	var requestData map[string]interface{}
	json.NewDecoder(r.Body).Decode(&requestData)

	invoiceID := requestData["invoiceID"].(string)
	response := map[string]string{
		"message": fmt.Sprintf("Invoice %s created or updated successfully!", invoiceID),
	}
	json.NewEncoder(w).Encode(response)
}

func GetTotalPurchases(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	itemID := vars["itemID"]

	response := map[string]string{
		"itemID":         itemID,
		"totalPurchases": "100",
	}
	json.NewEncoder(w).Encode(response)
}

func GetTotalSales(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	itemID := vars["itemID"]

	response := map[string]string{
		"itemID":     itemID,
		"totalSales": "200",
	}
	json.NewEncoder(w).Encode(response)
}

func GetIndices(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	storeID := vars["storeID"]

	response := map[string]interface{}{
		"storeID":      storeID,
		"wastageIndex": "10",
		"ethicsIndex":  "80",
		"qualityIndex": "90",
	}
	json.NewEncoder(w).Encode(response)
}

func InvalidateTransaction(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	itemID := vars["itemID"]

	response := map[string]string{
		"message": fmt.Sprintf("Transaction for item %s invalidated successfully!", itemID),
	}
	json.NewEncoder(w).Encode(response)
}
