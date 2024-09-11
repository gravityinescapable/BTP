package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"time"

	"github.com/hyperledger/fabric-contract-api-go/contractapi"
)

type SmartContract struct {
	contractapi.Contract
}

// Invoice structure
type Invoice struct {
	InvoiceID       string  `json:"invoice_id"`
	StoreID         string  `json:"store_id"`
	Date            string  `json:"date"`
	Items           []Item  `json:"items"`
	TotalAmount     float64 `json:"total_amount"`
	TransactionHash string  `json:"transaction_hash"`
	Timestamp       string  `json:"timestamp"`
	InvoiceType     string  `json:"invoice_type"` // 'purchase' or 'sales'
	PrevBlockHash   string  `json:"prev_block_hash"`
}

// Item structure
type Item struct {
	ItemID       string  `json:"item_id"`
	ItemName     string  `json:"item_name"`
	Quantity     float64 `json:"quantity"`
	PricePerUnit float64 `json:"price_per_unit"`
	TotalPrice   float64 `json:"total_price"`
	ExpiryDate   string  `json:"expiry_date"`
	InvoiceType  string  `json:"invoice_type"` // 'purchase' or 'sales'
}

// ItemKey structure
type ItemKey struct {
	ItemID     string `json:"item_id"`
	ExpiryDate string `json:"expiry_date"`
}

// WastageIndex structure
type WastageIndex struct {
	ItemKey       ItemKey `json:"item_key"`
	Wastage       float64 `json:"wastage"`
	TotalPurchase float64 `json:"total_purchase"`
	TotalSales    float64 `json:"total_sales"`
}

// QualityIndex structure
type QualityIndex struct {
	StoreID      string  `json:"store_id"`
	ItemKey      ItemKey `json:"item_key"`
	QualityIndex float64 `json:"quality_index"`
}

// StoreQuality structure
type StoreQuality struct {
	StoreID           string  `json:"store_id"`
	TotalQualityIndex float64 `json:"total_quality_index"`
	NumItemKeys       int     `json:"num_item_keys"`
}

// TransactionValidity structure
type TransactionValidity struct {
	StoreID             string  `json:"store_id"`
	ItemKey             ItemKey `json:"item_key"`
	ValidTransactions   int     `json:"valid_transactions"`
	InvalidTransactions int     `json:"invalid_transactions"`
}

// Create or update an invoice and recalculate indices
func (s *SmartContract) CreateOrUpdateInvoice(ctx contractapi.TransactionContextInterface, invoice Invoice) error {
	// Generate the hash of the current block
	currentBlockHash := generateBlockHash(invoice)
	invoice.TransactionHash = currentBlockHash

	// Retrieve the previous block hash for provenance
	prevBlockHash, err := ctx.GetStub().GetState(invoice.InvoiceID)
	if err != nil {
		return fmt.Errorf("failed to retrieve previous block hash: %s", err.Error())
	}
	if prevBlockHash != nil {
		invoice.PrevBlockHash = string(prevBlockHash)
	}

	// Validate transaction
	err = s.ValidateTransaction(ctx, invoice)
	if err != nil {
		return err
	}

	// Convert invoice to JSON and save to ledger
	invoiceJSON, err := json.Marshal(invoice)
	if err != nil {
		return err
	}

	err = ctx.GetStub().PutState(invoice.InvoiceID, invoiceJSON)
	if err != nil {
		return err
	}

	// Calculate wastage, quality, and ethics index
	wastageIndices, err := s.CalculateWastageIndex(ctx, invoice.StoreID, invoice.Items)
	if err != nil {
		return err
	}

	qualityIndex, err := s.CalculateQualityIndex(ctx, invoice.StoreID, wastageIndices)
	if err != nil {
		return err
	}

	ethicsIndex, err := s.CalculateEthicsIndex(ctx, invoice.StoreID, wastageIndices)
	if err != nil {
		return err
	}

	// Update ledger with new indices
	for _, wastageIndex := range wastageIndices {
		err = s.UpdateLedgerWithIndices(ctx, invoice.StoreID, qualityIndex, wastageIndex, ethicsIndex, TransactionValidity{})
		if err != nil {
			return err
		}
	}

	return nil
}

// Validate a transaction and flag it as invalid if necessary
func (s *SmartContract) ValidateTransaction(ctx contractapi.TransactionContextInterface, invoice Invoice) error {
	currentDate := time.Now().Format("2006-01-02")

	for _, item := range invoice.Items {
		// Check if the item has expired
		if currentDate > item.ExpiryDate {
			err := s.MarkTransactionInvalid(ctx, invoice.StoreID, ItemKey{ItemID: item.ItemID, ExpiryDate: item.ExpiryDate})
			if err != nil {
				return fmt.Errorf("transaction is invalid due to expired item: %s", err.Error())
			}
		}
		// Check if total sales exceed total purchases
		totalPurchases := s.GetTotalPurchases(ctx, invoice.StoreID, ItemKey{ItemID: item.ItemID, ExpiryDate: item.ExpiryDate})
		totalSales := s.GetTotalSales(ctx, invoice.StoreID, ItemKey{ItemID: item.ItemID, ExpiryDate: item.ExpiryDate})

		if totalSales > totalPurchases {
			err := s.MarkTransactionInvalid(ctx, invoice.StoreID, ItemKey{ItemID: item.ItemID, ExpiryDate: item.ExpiryDate})
			if err != nil {
				return fmt.Errorf("transaction is invalid due to sales exceeding purchases: %s", err.Error())
			}
		}
	}
	return nil
}

// Mark a transaction as invalid and delete it while maintaining provenance
func (s *SmartContract) MarkTransactionInvalid(ctx contractapi.TransactionContextInterface, storeID string, itemKey ItemKey) error {
	// Retrieve the invoice to be invalidated
	invoiceJSON, err := ctx.GetStub().GetState(itemKey.ItemID)
	if err != nil {
		return err
	}
	if invoiceJSON == nil {
		return fmt.Errorf("Invoice not found for ItemID: %s", itemKey.ItemID)
	}

	var invoice Invoice
	err = json.Unmarshal(invoiceJSON, &invoice)
	if err != nil {
		return err
	}

	// Update the transaction validity
	err = s.UpdateTransactionValidity(ctx, storeID, itemKey, false)
	if err != nil {
		return err
	}

	// Log the invalid transaction
	err = ctx.GetStub().PutState(fmt.Sprintf("INVALID_%s_%s_%s", storeID, itemKey.ItemID, itemKey.ExpiryDate), invoiceJSON)
	if err != nil {
		return err
	}

	// Delete the invoice while maintaining provenance
	err = ctx.GetStub().DelState(itemKey.ItemID)
	if err != nil {
		return err
	}

	return nil
}

// Update the validity of a transaction
func (s *SmartContract) UpdateTransactionValidity(ctx contractapi.TransactionContextInterface, storeID string, itemKey ItemKey, isValid bool) error {
	// Retrieve current validity data
	transactionValidityBytes, err := ctx.GetStub().GetState(fmt.Sprintf("TRANSACTION_VALIDITY_%s_%s_%s", storeID, itemKey.ItemID, itemKey.ExpiryDate))
	if err != nil {
		return err
	}

	var transactionValidity TransactionValidity
	if transactionValidityBytes != nil {
		err = json.Unmarshal(transactionValidityBytes, &transactionValidity)
		if err != nil {
			return err
		}
	} else {
		transactionValidity = TransactionValidity{
			StoreID: storeID,
			ItemKey: itemKey,
		}
	}

	// Update the count
	if isValid {
		transactionValidity.ValidTransactions++
	} else {
		transactionValidity.InvalidTransactions++
	}

	// Save updated validity data on the ledger
	transactionValidityBytes, err = json.Marshal(transactionValidity)
	if err != nil {
		return err
	}

	err = ctx.GetStub().PutState(fmt.Sprintf("TRANSACTION_VALIDITY_%s_%s_%s", storeID, itemKey.ItemID, itemKey.ExpiryDate), transactionValidityBytes)
	if err != nil {
		return err
	}

	return nil
}

// Calculate wastage index for given items
func (s *SmartContract) CalculateWastageIndex(ctx contractapi.TransactionContextInterface, storeID string, items []Item) ([]WastageIndex, error) {
	var wastageIndices []WastageIndex

	for _, item := range items {
		itemKey := ItemKey{ItemID: item.ItemID, ExpiryDate: item.ExpiryDate}

		// Fetch all purchase and sales transactions related to this itemKey
		totalPurchases := s.GetTotalPurchases(ctx, storeID, itemKey)
		totalSales := s.GetTotalSales(ctx, storeID, itemKey)

		wastage := totalPurchases - totalSales

		if totalSales > totalPurchases {
			// Mark transactions as invalid
			s.MarkTransactionInvalid(ctx, storeID, itemKey)
		}

		wastageIndex := WastageIndex{
			ItemKey:       itemKey,
			Wastage:       wastage / totalPurchases * 100,
			TotalPurchase: totalPurchases,
			TotalSales:    totalSales,
		}

		wastageIndices = append(wastageIndices, wastageIndex)
	}

	return wastageIndices, nil
}

// Calculate quality index based on wastage index
func (s *SmartContract) CalculateQualityIndex(ctx contractapi.TransactionContextInterface, storeID string, wastageIndices []WastageIndex) (float64, error) {
	var totalWastageIndex float64
	for _, wastageIndex := range wastageIndices {
		totalWastageIndex += wastageIndex.Wastage
	}

	// Quality index = 1/wastage index
	averageWastageIndex := totalWastageIndex / float64(len(wastageIndices))
	qualityIndex := 1 / averageWastageIndex

	return qualityIndex, nil
}

// Calculate ethics index for the store
func (s *SmartContract) CalculateEthicsIndex(ctx contractapi.TransactionContextInterface, storeID string, wastageIndices []WastageIndex) (float64, error) {
	var totalValidTransactions, totalInvalidTransactions int

	for _, wastageIndex := range wastageIndices {
		itemKey := wastageIndex.ItemKey
		transactionValidity, err := s.GetTransactionValidity(ctx, storeID, itemKey)
		if err != nil {
			return 0, err
		}

		totalValidTransactions += transactionValidity.ValidTransactions
		totalInvalidTransactions += transactionValidity.InvalidTransactions
	}

	// Ethics index = valid / (valid + invalid)
	averageethicsIndex := float64(totalValidTransactions) / float64(totalValidTransactions+totalInvalidTransactions) * 100

	return averageethicsIndex, nil
}

// Updates the ledger with calculated indices
func (s *SmartContract) UpdateLedgerWithIndices(ctx contractapi.TransactionContextInterface, storeID string, qualityIndex float64, wastageIndex WastageIndex, averageethicsIndex float64, transactionValidity TransactionValidity) error {
	// Update quality index in ledger
	qualityIndexKey := fmt.Sprintf("QUALTIY_INDEX_%s_%s_%s", storeID, wastageIndex.ItemKey.ItemID, wastageIndex.ItemKey.ExpiryDate)
	qualityIndexData := QualityIndex{
		StoreID:      storeID,
		QualityIndex: qualityIndex + averageethicsIndex,
	}
	qualityIndexJSON, err := json.Marshal(qualityIndexData)
	if err != nil {
		return err
	}
	err = ctx.GetStub().PutState(qualityIndexKey, qualityIndexJSON)
	if err != nil {
		return err
	}

	// Update wastage index in ledger
	wastageIndexKey := fmt.Sprintf("WASTAGE_INDEX_%s_%s_%s", storeID, wastageIndex.ItemKey.ItemID, wastageIndex.ItemKey.ExpiryDate)
	wastageIndexJSON, err := json.Marshal(wastageIndex)
	if err != nil {
		return err
	}
	err = ctx.GetStub().PutState(wastageIndexKey, wastageIndexJSON)
	if err != nil {
		return err
	}

	// Update transaction validity in ledger
	transactionValidityKey := fmt.Sprintf("TRANSACTION_VALIDITY_%s_%s_%s", storeID, wastageIndex.ItemKey.ItemID, wastageIndex.ItemKey.ExpiryDate)
	transactionValidityJSON, err := json.Marshal(transactionValidity)
	if err != nil {
		return err
	}
	err = ctx.GetStub().PutState(transactionValidityKey, transactionValidityJSON)
	if err != nil {
		return err
	}

	return nil
}

// Delete an invoice and maintain provenance
func (s *SmartContract) DeleteInvoice(ctx contractapi.TransactionContextInterface, invoiceID string) error {
	// Retrieve the invoice to be deleted
	invoiceJSON, err := ctx.GetStub().GetState(invoiceID)
	if err != nil {
		return err
	}
	if invoiceJSON == nil {
		return fmt.Errorf("Invoice not found for ID: %s", invoiceID)
	}

	var invoice Invoice
	err = json.Unmarshal(invoiceJSON, &invoice)
	if err != nil {
		return err
	}

	// Delete the invoice from ledger
	err = ctx.GetStub().DelState(invoiceID)
	if err != nil {
		return err
	}

	// Log the deletion for provenance
	err = ctx.GetStub().PutState(fmt.Sprintf("DELETED_%s_%s", invoice.StoreID, invoiceID), invoiceJSON)
	if err != nil {
		return err
	}

	return nil
}

// Update an existing invoice and recalculate indices
func (s *SmartContract) UpdateInvoice(ctx contractapi.TransactionContextInterface, invoice Invoice) error {
	// Retrieve the current invoice to be updated
	existingInvoiceJSON, err := ctx.GetStub().GetState(invoice.InvoiceID)
	if err != nil {
		return err
	}
	if existingInvoiceJSON == nil {
		return fmt.Errorf("Invoice not found for ID: %s", invoice.InvoiceID)
	}

	var existingInvoice Invoice
	err = json.Unmarshal(existingInvoiceJSON, &existingInvoice)
	if err != nil {
		return err
	}

	// Delete the existing invoice while maintaining provenance
	err = s.DeleteInvoice(ctx, invoice.InvoiceID)
	if err != nil {
		return err
	}

	// Create or update the invoice with the new data
	err = s.CreateOrUpdateInvoice(ctx, invoice)
	if err != nil {
		return err
	}

	return nil
}

// Retrieve total purchases for a specific itemkey
func (s *SmartContract) GetTotalPurchases(ctx contractapi.TransactionContextInterface, storeID string, itemKey ItemKey) float64 {
	queryString := fmt.Sprintf(`{"selector":{"store_id":"%s","items":{"$elemMatch":{"item_id":"%s","expiry_date":"%s"}},"invoice_type":"purchase"}}`, storeID, itemKey.ItemID, itemKey.ExpiryDate)
	resultsIterator, err := ctx.GetStub().GetQueryResult(queryString)
	if err != nil {
		return -1
	}
	defer resultsIterator.Close()

	var totalPurchases float64
	for resultsIterator.HasNext() {
		queryResponse, err := resultsIterator.Next()
		if err != nil {
			return -1
		}

		var invoice Invoice
		err = json.Unmarshal(queryResponse.Value, &invoice)
		if err != nil {
			return -1
		}

		for _, item := range invoice.Items {
			if item.ItemID == itemKey.ItemID && item.ExpiryDate == itemKey.ExpiryDate {
				totalPurchases += item.Quantity
			}
		}
	}

	return totalPurchases
}

// Retrieve total sales for a specific itemkey
func (s *SmartContract) GetTotalSales(ctx contractapi.TransactionContextInterface, storeID string, itemKey ItemKey) float64 {
	queryString := fmt.Sprintf(`{"selector":{"store_id":"%s","items":{"$elemMatch":{"item_id":"%s","expiry_date":"%s"}},"invoice_type":"sales"}}`, storeID, itemKey.ItemID, itemKey.ExpiryDate)
	resultsIterator, err := ctx.GetStub().GetQueryResult(queryString)
	if err != nil {
		return -1
	}
	defer resultsIterator.Close()

	var totalSales float64
	for resultsIterator.HasNext() {
		queryResponse, err := resultsIterator.Next()
		if err != nil {
			return -1
		}

		var invoice Invoice
		err = json.Unmarshal(queryResponse.Value, &invoice)
		if err != nil {
			return -1
		}

		for _, item := range invoice.Items {
			if item.ItemID == itemKey.ItemID && item.ExpiryDate == itemKey.ExpiryDate {
				totalSales += item.Quantity
			}
		}
	}

	return totalSales
}

// Retrieve transaction validity data from the ledger
func (s *SmartContract) GetTransactionValidity(ctx contractapi.TransactionContextInterface, storeID string, itemKey ItemKey) (TransactionValidity, error) {
	transactionValidityBytes, err := ctx.GetStub().GetState(fmt.Sprintf("TRANSACTION_VALIDITY_%s_%s_%s", storeID, itemKey.ItemID, itemKey.ExpiryDate))
	if err != nil {
		return TransactionValidity{}, err
	}
	if transactionValidityBytes == nil {
		return TransactionValidity{}, fmt.Errorf("transaction validity not found for ItemKey: %s", itemKey)
	}

	var transactionValidity TransactionValidity
	err = json.Unmarshal(transactionValidityBytes, &transactionValidity)
	if err != nil {
		return TransactionValidity{}, err
	}

	return transactionValidity, nil
}

// Generate a SHA-256 hash for the block
func generateBlockHash(invoice Invoice) string {
	record := invoice.InvoiceID + invoice.StoreID + invoice.Date + invoice.Timestamp
	hash := sha256.New()
	hash.Write([]byte(record))
	return hex.EncodeToString(hash.Sum(nil))
}

// Calculate rewards or corrective measures based on quality index
func (s *SmartContract) RewardAndCorrectiveSystem(ctx contractapi.TransactionContextInterface, storeID string, qualityIndex float64) (float64, error) {
	var Cs, Rs float64

	// Retrieve the corrective coefficient and reward coefficient
	Cs, err := s.CalculateCorrectiveCoefficient(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to calculate corrective coefficient: %s", err.Error())
	}

	Rs, err = s.CalculateRewardCoefficient(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to calculate reward coefficient: %s", err.Error())
	}

	var result float64

	// Calculate corrective measure or reward based on quality index
	if qualityIndex < 50 {
		result = -Cs * (50 - qualityIndex) // Corrective measure
	} else if qualityIndex >= 80 {
		result = Rs * (qualityIndex - 50) // Reward
	} else {
		result = 0 // Neutral zone
	}

	return result, nil
}

// Calculate the corrective coefficient based on quality index values
func (s *SmartContract) CalculateCorrectiveCoefficient(ctx contractapi.TransactionContextInterface) (float64, error) {
	query := `{"selector": {"quality_index": {"$lte": 50}}}`
	resultsIterator, err := ctx.GetStub().GetQueryResult(query)
	if err != nil {
		return 0, err
	}
	defer resultsIterator.Close()

	var minQualityIndex, maxQualityIndex float64
	minQualityIndex = math.MaxFloat64
	maxQualityIndex = -math.MaxFloat64

	for resultsIterator.HasNext() {
		response, err := resultsIterator.Next()
		if err != nil {
			return 0, err
		}

		var record struct {
			QualityIndex float64 `json:"quality_index"`
		}
		err = json.Unmarshal(response.Value, &record)
		if err != nil {
			return 0, err
		}

		if record.QualityIndex < minQualityIndex {
			minQualityIndex = record.QualityIndex
		}
		if record.QualityIndex > maxQualityIndex {
			maxQualityIndex = record.QualityIndex
		}
	}

	if minQualityIndex == 0 {
		return maxQualityIndex, nil
	}

	return maxQualityIndex / minQualityIndex, nil
}

// Calculate the reward coefficient based on quality index values
func (s *SmartContract) CalculateRewardCoefficient(ctx contractapi.TransactionContextInterface) (float64, error) {
	query := `{"selector": {"quality_index": {"$gte": 80}}}`
	resultsIterator, err := ctx.GetStub().GetQueryResult(query)
	if err != nil {
		return 0, err
	}
	defer resultsIterator.Close()

	var minQualityIndex, maxQualityIndex float64
	minQualityIndex = math.MaxFloat64
	maxQualityIndex = -math.MaxFloat64

	for resultsIterator.HasNext() {
		response, err := resultsIterator.Next()
		if err != nil {
			return 0, err
		}

		var record struct {
			QualityIndex float64 `json:"quality_index"`
		}
		err = json.Unmarshal(response.Value, &record)
		if err != nil {
			return 0, err
		}

		if record.QualityIndex < minQualityIndex {
			minQualityIndex = record.QualityIndex
		}
		if record.QualityIndex > maxQualityIndex {
			maxQualityIndex = record.QualityIndex
		}
	}

	if minQualityIndex == math.MaxFloat64 {
		// No data in this range
		return 0, nil
	}

	return maxQualityIndex / minQualityIndex, nil
}

func main() {
	chaincode, err := contractapi.NewChaincode(new(SmartContract))
	if err != nil {
		fmt.Printf("Error creating invoice chaincode: %s", err.Error())
		return
	}

	if err := chaincode.Start(); err != nil {
		fmt.Printf("Error starting invoice chaincode: %s", err.Error())
	}
}
