const axios = require('axios');

// Function to handle POS data
async function handlePOSData(data) {
    // Process each item and format it using the itemData structure
    const formattedItems = data.items.map(item => {
        return {
            itemID: item.itemID,
            itemName: item.itemName || 'Unknown Item',
            quantity: item.quantity || 1,
            pricePerUnit: item.pricePerUnit || 0.0,
            totalPrice: (item.quantity || 1) * (item.pricePerUnit || 0.0),
            expiryDate: item.expiryDate || 'N/A',
            storeID: 'STORE001', 
            eventType: 'purchase',
            timestamp: new Date().toISOString() 
        };
    });

    const posData = {
        transactionID: data.transactionID,
        items: formattedItems, 
        totalAmount: data.totalAmount,
        timestamp: new Date().toISOString(),
        storeID: 'STORE001', 
    };

    try {
        const response = await axios.post('http://pos-system-url/api/transactions', posData);
        console.log('Transaction recorded:', response.data);
    } catch (error) {
        console.error('Error sending data to POS system:', error);
    }
}

// Data received from POS
const exampleData = {
    transactionID: '12345',
    items: [
        { itemID: 'item01', itemName: 'Item Name 1', quantity: 2, pricePerUnit: 10.0 },
        { itemID: 'item02', itemName: 'Item Name 2', quantity: 1, pricePerUnit: 20.0 }
    ],
    totalAmount: 40.0,
};

// Handle the data
handlePOSData(exampleData);
