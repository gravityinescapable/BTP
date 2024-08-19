package main

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/hyperledger/fabric-contract-api-go/contractapi"
)

// Converts a Go object to a JSON string
func SerializeToJSON(data interface{}) ([]byte, error) {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("Error serializing data to JSON: %v", err)
	}
	return jsonData, nil
}

// Converts a JSON string to a Go object
func DeserializeFromJSON(jsonData []byte, v interface{}) error {
	err := json.Unmarshal(jsonData, v)
	if err != nil {
		return fmt.Errorf("Error deserializing JSON data: %v", err)
	}
	return nil
}

// Creates a provenance record in the ledger
func CreateProvenanceRecord(existingInvoiceJSON []byte, provenanceRecordID string, ctx contractapi.TransactionContextInterface) error {

	// Store the provenance record in the ledger
	return ctx.GetStub().PutState(provenanceRecordID, existingInvoiceJSON)
}

// Ensures the invoice type is either "purchase" or "sales"
func ValidateInvoiceType(invoiceType string) error {
	if invoiceType != "purchase" && invoiceType != "sales" {
		return fmt.Errorf("Invalid invoice type %s, must be 'purchase' or 'sales'", invoiceType)
	}
	return nil
}

// Checks if the item is valid (i.e., is a food item)
func ValidateItem(item Item) error {
	if !item.IsFoodItem {
		return fmt.Errorf("Item %s is not a food item", item.ItemID)
	}
	return nil
}

// Parses and validates a date string according to the given layout
func ValidateDateFormat(dateStr string, layout string) (time.Time, error) {
	parsedDate, err := time.Parse(layout, dateStr)
	if err != nil {
		return time.Time{}, fmt.Errorf("Invalid date format: %v", err)
	}
	return parsedDate, nil
}
