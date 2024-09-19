const RFID = require('node-rfid'); 
const axios = require('axios');

// Initialize RFID Reader
const rfid = new RFID('/dev/tty-usbserial1', { baudRate: 9600 });

// Event listener for detecting tags 
rfid.on('data', (data) => {
    const itemData = {
        itemID: data.id,                     
        itemName: data.name || 'Unknown Item',                 
        quantity: data.quantity || 1,             
        pricePerUnit: data.pricePerUnit || 0.0,    
        totalPrice: data.totalPrice || 0.0,         
        expiryDate: data.expiryDate || 'N/A',         
        storeID: 'STORE001',                 
        eventType: 'purchase',               
        timestamp: new Date().toISOString()  
    };

    // Send data to middleware system
    axios.post('http://middleware-system-url/api/storeItem', itemData)
        .then(response => {
            console.log('Item recorded on blockchain: ', response.data);
        })
        .catch(error => {
            console.error('Error sending data: ', error);
        });
});

// Handle RFID Reader errors
rfid.on('error', (error) => {
    console.error('RFID Read Error:', error);
});


