package main

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/hyperledger/fabric-contract-api-go/contractapi"
)

// Checks if the invoice type is valid
func ValidateInvoiceType(invoiceType string) error {
	if invoiceType != "purchase" && invoiceType != "sales" {
		return fmt.Errorf("Invoice type must be either 'purchase' or 'sales'")
	}
	return nil
}

// Checks if the item is valid
func ValidateItem(item Item) error {
	if !item.IsFoodItem {
		return fmt.Errorf("Only food items can be recorded in the invoice")
	}
	return nil
}

// Parses and validates date formats
func ValidateDateFormat(dateStr string, layout string) (time.Time, error) {
	date, err := time.Parse(layout, dateStr)
	if err != nil {
		return time.Time{}, fmt.Errorf("Invalid date format: %v", err)
	}
	return date, nil
}

// Converts a struct to JSON
func SerializeToJSON(v interface{}) ([]byte, error) {
	return json.Marshal(v)
}

// Converts JSON to a struct
func DeserializeFromJSON(data []byte, v interface{}) error {
	return json.Unmarshal(data, v)
}

// Creates and stores a provenance record
func CreateProvenanceRecord(existingData []byte, recordID string, ctx contractapi.TransactionContextInterface) error {
	provenanceRecord := struct {
		PreviousState string `json:"previous_state"`
		Timestamp     string `json:"timestamp"`
	}{
		PreviousState: string(existingData),
		Timestamp:     time.Now().String(),
	}
	provenanceJSON, err := SerializeToJSON(provenanceRecord)
	if err != nil {
		return err
	}
	return ctx.GetStub().PutState(recordID, provenanceJSON)
}

// Get counts of valid and invalid transactions
func GetTransactionCounts(ctx contractapi.TransactionContextInterface, itemID string) (int, int, error) {
	queryString := fmt.Sprintf(`{"selector":{"item_id":"%s"}}`, itemID)
	resultsIterator, err := ctx.GetStub().GetQueryResult(queryString)
	if err != nil {
		return 0, 0, err
	}
	defer resultsIterator.Close()

	var invalidCount int
	var totalCount int

	for resultsIterator.HasNext() {
		queryResponse, err := resultsIterator.Next()
		if err != nil {
			return 0, 0, err
		}

		var transaction struct {
			IsValid bool `json:"is_valid"`
		}

		err = DeserializeFromJSON(queryResponse.Value, &transaction)
		if err != nil {
			return 0, 0, err
		}

		if !transaction.IsValid {
			invalidCount++
		}
		totalCount++
	}

	return invalidCount, totalCount, nil
}
