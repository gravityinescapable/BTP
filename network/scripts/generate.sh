#!/bin/bash

# Clean up the previous crypto material and config transactions
rm -rf crypto-config
rm -rf *.block
rm -rf *.tx

# Generate Crypto Material
cryptogen generate --config=./crypto-config.yaml

# Generate Genesis Block
configtxgen -profile TwoOrgsOrdererGenesis -outputBlock ./genesis.block

# Generate Channel Configuration Transaction
configtxgen -profile TwoOrgsChannel -outputCreateChannelTx ./channel.tx -channelID mychannel

# Generate Anchor Peer Transaction
configtxgen -profile TwoOrgsChannel -outputAnchorPeersUpdate ./Org1MSPanchors.tx -channelID mychannel -asOrg Org1MSP
