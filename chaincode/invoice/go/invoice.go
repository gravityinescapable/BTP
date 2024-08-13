package main

import (
	"fmt"
	"time"

	"github.com/hyperledger/fabric-contract-api-go/contractapi"
)

// Structure of an invoice
type Invoice struct {
	InvoiceID       string  `json:"invoice_id"`
	StoreID         string  `json:"store_id"`
	Date            string  `json:"date"`
	Items           []Item  `json:"items"`
	TotalAmount     float64 `json:"total_amount"`
	TransactionHash string  `json:"transaction_hash"`
	Timestamp       string  `json:"timestamp"`
	InvoiceType     string  `json:"invoice_type"` // 'purchase' or 'sales'
}

// Structure of an item within an invoice
type Item struct {
	ItemID       string  `json:"item_id"`
	ItemName     string  `json:"item_name"`
	Quantity     int     `json:"quantity"`
	PricePerUnit float64 `json:"price_per_unit"`
	TotalPrice   float64 `json:"total_price"`
	ExpiryDate   string  `json:"expiry_date"`
	IsFoodItem   bool    `json:"is_food_item"`
	InvoiceType  string  `json:"invoice_type"` // 'purchase' or 'sales'
}

// SmartContract to manage invoices
type SmartContract struct {
	contractapi.Contract
}

// Creates a new invoice and stores it in the ledger
func (s *SmartContract) CreateInvoice(ctx contractapi.TransactionContextInterface, invoiceID string, storeID string, date string, invoiceType string, items []Item, totalAmount float64) error {
	// Validate the invoice type
	if err := ValidateInvoiceType(invoiceType); err != nil {
		return err
	}

	// Validate each item in the invoice
	for _, item := range items {
		if err := ValidateItem(item); err != nil {
			return err
		}

		// Parse the expiry date of the item
		expiryDate, err := ValidateDateFormat(item.ExpiryDate, "2006-01-02")
		if err != nil {
			return fmt.Errorf("Invalid expiry date format for item %s: %v", item.ItemID, err)
		}

		// Parse the transaction date
		transactionDate, err := ValidateDateFormat(date, "2006-01-02")
		if err != nil {
			return fmt.Errorf("Invalid transaction date format: %v", err)
		}

		// Check if the item is expired at the time of the transaction
		if transactionDate.After(expiryDate) {
			return fmt.Errorf("Transaction cannot be recorded because item %s has expired", item.ItemID)
		}

		// Add invoice type to the item
		item.InvoiceType = invoiceType
	}

	// Create the invoice object
	invoice := Invoice{
		InvoiceID:   invoiceID,
		StoreID:     storeID,
		Date:        date,
		Items:       items,
		TotalAmount: totalAmount,
		Timestamp:   time.Now().String(),
		InvoiceType: invoiceType,
	}

	// Serialize the invoice object to JSON
	invoiceJSON, err := SerializeToJSON(invoice)
	if err != nil {
		return err
	}

	// Store the invoice in the ledger
	return ctx.GetStub().PutState(invoice.InvoiceID, invoiceJSON)
}

// Updates an existing invoice and maintains provenance
func (s *SmartContract) UpdateInvoice(ctx contractapi.TransactionContextInterface, invoiceID string, updatedInvoice Invoice) error {
	// Retrieve the existing invoice
	existingInvoiceJSON, err := ctx.GetStub().GetState(invoiceID)
	if err != nil {
		return fmt.Errorf("Failed to read from world state: %v", err)
	}
	if existingInvoiceJSON == nil {
		return fmt.Errorf("Invoice %s does not exist", invoiceID)
	}

	// Create a provenance record for the existing invoice
	provenanceRecordID := fmt.Sprintf("%s_provenance_%s", invoiceID, time.Now().Format("20060102150405"))
	if err := CreateProvenanceRecord(existingInvoiceJSON, provenanceRecordID, ctx); err != nil {
		return err
	}

	// Serialize the updated invoice object to JSON
	updatedInvoiceJSON, err := SerializeToJSON(updatedInvoice)
	if err != nil {
		return err
	}

	// Store the updated invoice in the ledger
	return ctx.GetStub().PutState(invoiceID, updatedInvoiceJSON)
}

// Deletes an invoice from the ledger and maintains provenance
func (s *SmartContract) DeleteInvoice(ctx contractapi.TransactionContextInterface, invoiceID string) error {
	// Retrieve the existing invoice
	existingInvoiceJSON, err := ctx.GetStub().GetState(invoiceID)
	if err != nil {
		return fmt.Errorf("Failed to read from world state: %v", err)
	}
	if existingInvoiceJSON == nil {
		return fmt.Errorf("Invoice %s does not exist", invoiceID)
	}

	// Create a provenance record for the existing invoice
	provenanceRecordID := fmt.Sprintf("%s_provenance_%s", invoiceID, time.Now().Format("20060102150405"))
	if err := CreateProvenanceRecord(existingInvoiceJSON, provenanceRecordID, ctx); err != nil {
		return err
	}

	// Delete the invoice from the ledger
	return ctx.GetStub().DelState(invoiceID)
}

// Calculate wastage for an item within a rolling window
func (s *SmartContract) CalculateWastageInRollingWindow(ctx contractapi.TransactionContextInterface, itemID string, expiryDate string) (float64, error) {
	// Composite key
	itemKey := struct {
		ItemID     string `json:"item_id"`
		ExpiryDate string `json:"expiry_date"`
	}{
		ItemID:     itemID,
		ExpiryDate: expiryDate,
	}

	// Parse the expiry date
	expiryDateParsed, err := ValidateDateFormat(expiryDate, "2006-01-02")
	if err != nil {
		return 0, fmt.Errorf("Invalid expiry date format: %v", err)
	}

	// Fetch all invoices associated with an itemKey
	queryString := fmt.Sprintf(`{"selector":{"items.item_id":"%s","items.expiry_date":"%s"}}`, itemID, expiryDate)
	resultsIterator, err := ctx.GetStub().GetQueryResult(queryString)
	if err != nil {
		return 0, fmt.Errorf("Failed to query ledger: %v", err)
	}
	defer resultsIterator.Close()

	var firstPurchaseDate time.Time
	isFirstPurchaseFound := false

	// Identify the first purchase date
	for resultsIterator.HasNext() {
		queryResponse, err := resultsIterator.Next()
		if err != nil {
			return 0, err
		}

		var invoice Invoice
		err = DeserializeFromJSON(queryResponse.Value, &invoice)
		if err != nil {
			return 0, err
		}

		// Parse the invoice date
		invoiceDateParsed, err := ValidateDateFormat(invoice.Date, "2006-01-02")
		if err != nil {
			return 0, fmt.Errorf("Invalid invoice date format: %v", err)
		}

		for _, item := range invoice.Items {
			if item.ItemID == itemID && item.ExpiryDate == expiryDate && invoice.InvoiceType == "purchase" {
				// Check and set the first purchase date
				if !isFirstPurchaseFound || invoiceDateParsed.Before(firstPurchaseDate) {
					firstPurchaseDate = invoiceDateParsed
					isFirstPurchaseFound = true
					break
				}
			}
		}
	}

	if !isFirstPurchaseFound {
		return 0, fmt.Errorf("No purchase data found for item %s", itemID)
	}

	// Calculate wastage based on the first purchase date and current date
	totalSales := 0.0
	totalPurchases := 0.0
	currentDate := time.Now()
	windowStartDate := firstPurchaseDate

	// Query for all sales and purchases in the window
	resultsIterator, err := ctx.GetStub().GetQueryResult(queryString)
	if err != nil {
		return 0, fmt.Errorf("Failed to query ledger: %v", err)
	}
	defer resultsIterator.Close()

	for resultsIterator.HasNext() {
		queryResponse, err := resultsIterator.Next()
		if err != nil {
			return 0, err
		}

		var invoice Invoice
		err = DeserializeFromJSON(queryResponse.Value, &invoice)
		if err != nil {
			return 0, err
		}

		// Parse the invoice date
		invoiceDateParsed, err := ValidateDateFormat(invoice.Date, "2006-01-02")
		if err != nil {
			return 0, fmt.Errorf("Invalid invoice date format: %v", err)
		}

		for _, item := range invoice.Items {
			if item.ItemID == itemID && item.ExpiryDate == expiryDate {
				if invoice.InvoiceType == "purchase" {
					totalPurchases += item.Quantity
				} else if invoice.InvoiceType == "sales" {
					totalSales += item.Quantity
				}
			}
		}
	}

	// Calculate wastage
	wastage := totalPurchases - totalSales
	return wastage, nil
}

// Driver function for the chaincode
func main() {
	chaincode, err := contractapi.NewChaincode(new(SmartContract))
	if err != nil {
		fmt.Printf("Error creating chaincode: %v\n", err)
		return
	}

	if err := chaincode.Start(); err != nil {
		fmt.Printf("Error starting chaincode: %v\n", err)
	}
}
