#!/bin/bash

# Bring up the network
docker-compose -f docker-compose.yaml up -d

# Wait for the containers to start
sleep 10

# Create the channel
docker exec -it cli peer channel create -o orderer.example.com:7050 -c mychannel -f ./channel.tx

# Join peer0.org1.example.com to the channel
docker exec -it cli peer channel join -b mychannel.block

# Update anchor peers
docker exec -it cli peer channel update -o orderer.example.com:7050 -c mychannel -f ./Org1MSPanchors.tx
