#!/bin/bash

# Generate the crypto-config directory
cryptogen generate --config=../crytpo-config.yaml --output="../crypto-config"

# Generate the channel configuration transaction
configtxgen -profile TwoOrgsOrdererGenesis -outputBlock channel-artifacts/genesis.block

# Generate the channel configuration transaction
configtxgen -profile TwoOrgsChannel -outputCreateChannelTx channel-artifacts/channel.tx -channelID mychannel
