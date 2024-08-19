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

// Updates an existing invoice
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

// Deletes an invoice from the ledger
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
	resultsIterator, err = ctx.GetStub().GetQueryResult(queryString)
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

		// Check if the transaction date exceeds the expiry date
		transactionTime, err := time.Parse(time.RFC3339, invoice.Date)
		if err != nil {
			return fmt.Errorf("Error parsing transaction date: %v", err)
		}

		expiryTime, err := time.Parse(time.RFC3339, expiryDate)
		if err != nil {
			return fmt.Errorf("Error parsing expiry date: %v", err)
		}

		// Invalidate the transaction if the transaction date exceeds the expiry date
		if transactionTime.After(expiryTime) {
			err = ctx.GetStub().DelState(invoice.InvoiceID)
			if err != nil {
				return fmt.Errorf("Failed to delete invoice %s: %v", invoice.InvoiceID, err)
			}
			continue
		}

		for _, item := range invoice.Items {
			if item.ItemID == itemID && item.ExpiryDate == expiryDate {
				if invoice.InvoiceType == "sales" {
					totalSales += float64(item.Quantity)
				} else if invoice.InvoiceType == "purchase" {
					totalPurchases += float64(item.Quantity)
				}
			}
		}
	}

	// Ensure total sales do not exceed total purchases
	if totalSales > totalPurchases {
		err := s.InvalidateTransactions(ctx, itemID, expiryDate)
		if err != nil {
			return 0, err
		}
		return 0, fmt.Errorf("Total sales exceed total purchases for item %s with expiry date %s. All related transactions have been invalidated.", itemID, expiryDate)
	}

	wastage := totalPurchases - totalSales

	// Return the calculated wastage
	return wastage, nil
}

// Calculate the quality index of a store
func (s *SmartContract) CalculateQualityIndex(ctx contractapi.TransactionContextInterface, storeID string) (float64, error) {

	// Fetch all invoices associated with the storeID
	queryString := fmt.Sprintf(`{"selector":{"store_id":"%s"}}`, storeID)
	resultsIterator, err := ctx.GetStub().GetQueryResult(queryString)
	if err != nil {
		return 0, fmt.Errorf("Failed to query ledger: %v", err)
	}
	defer resultsIterator.Close()

	var totalQualityIndex float64
	var itemKeyCount int

	// Iterate through the results and calculate the quality index for each item
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

		for _, item := range invoice.Items {
			// Calculate wastage
			wastage, err := s.CalculateWastageInRollingWindow(ctx, item.ItemID, item.ExpiryDate)
			if err != nil {
				return 0, err
			}

			// Calculate wastage index and ethics index
			wastageIndex := BoundIndex((wastage / float64(item.Quantity)) * 100)
			ethicsIndex, err := s.CalculateEthicsIndex(ctx, item.ItemID, item.ExpiryDate)
			if err != nil {
				return 0, err
			}

			// Calculate the quality index for the item
			qualityIndex := BoundIndex((1 / wastageIndex) + ethicsIndex)
			totalQualityIndex += qualityIndex
			itemKeyCount++
		}
	}

	// Return the average quality index for the store
	if itemKeyCount == 0 {
		return 0, fmt.Errorf("No items found for store %s", storeID)
	}

	return totalQualityIndex / float64(itemKeyCount), nil
}

// Ensure that index values are between 0 and 100
func BoundIndex(index float64) float64 {
	if index < 0 {
		return 0
	}
	if index > 100 {
		return 100
	}
	return index
}

// Invalidate all transactions related to a specific itemID and expiry date
func (s *SmartContract) InvalidateTransactions(ctx contractapi.TransactionContextInterface, itemID string, expiryDate string) error {

	// Fetch all invoices related to the itemID and expiryDate
	queryString := fmt.Sprintf(`{"selector":{"items.item_id":"%s","items.expiry_date":"%s"}}`, itemID, expiryDate)
	resultsIterator, err := ctx.GetStub().GetQueryResult(queryString)
	if err != nil {
		return fmt.Errorf("Failed to query ledger: %v", err)
	}
	defer resultsIterator.Close()

	// Iterate through the results and delete each invoice
	for resultsIterator.HasNext() {
		queryResponse, err := resultsIterator.Next()
		if err != nil {
			return err
		}

		var invoice Invoice
		err = DeserializeFromJSON(queryResponse.Value, &invoice)
		if err != nil {
			return err
		}

		err = ctx.GetStub().DelState(invoice.InvoiceID)
		if err != nil {
			return fmt.Errorf("Failed to delete invoice %s: %v", invoice.InvoiceID, err)
		}
	}

	return nil
}

// Calculate the ethics index of an item based on valid and invalid transactions
func (s *SmartContract) CalculateEthicsIndex(ctx contractapi.TransactionContextInterface, itemID string, expiryDate string) (float64, error) {

	// Fetch all invoices related to the itemID and expiryDate
	queryString := fmt.Sprintf(`{"selector":{"items.item_id":"%s","items.expiry_date":"%s"}}`, itemID, expiryDate)
	resultsIterator, err := ctx.GetStub().GetQueryResult(queryString)
	if err != nil {
		return 0, fmt.Errorf("Failed to query ledger: %v", err)
	}
	defer resultsIterator.Close()

	var totalValidTransactions, totalInvalidTransactions int

	// Iterate through the results and count valid and invalid transactions
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

		for _, item := range invoice.Items {
			if item.ItemID == itemID && item.ExpiryDate == expiryDate {
				transactionTime, err := time.Parse(time.RFC3339, invoice.Date)
				if err != nil {
					return 0, fmt.Errorf("Error parsing transaction date: %v", err)
				}

				expiryTime, err := time.Parse(time.RFC3339, expiryDate)
				if err != nil {
					return 0, fmt.Errorf("Error parsing expiry date: %v", err)
				}

				if transactionTime.After(expiryTime) {
					totalInvalidTransactions++
				} else {
					totalValidTransactions++
				}
			}
		}
	}

	if totalInvalidTransactions == 0 {
		return 100, nil
	}

	ethicsIndex := BoundIndex(float64(totalValidTransactions) / float64(totalInvalidTransactions))
	return ethicsIndex, nil
}

// Driver code
func main() {
	chaincode, err := contractapi.NewChaincode(&SmartContract{})
	if err != nil {
		fmt.Printf("Error creating smart contract: %v\n", err)
		return
	}

	if err := chaincode.Start(); err != nil {
		fmt.Printf("Error starting smart contract: %v\n", err)
	}
}
