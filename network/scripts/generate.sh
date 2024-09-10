#!/bin/bash

CONFIG_PATH="../configtx.yaml"
CRYPTO_PATH="../crytpo-config.yaml"
OUTPUT_PATH="../crypto-config"
ARTIFACTS_PATH="../channel-artifacts"  

# Create the artifacts directory
mkdir -p ${ARTIFACTS_PATH}

# Generate the crypto-config directory
cryptogen generate --config=${CRYPTO_PATH} --output=${OUTPUT_PATH}

# Generate the Genesis block for the orderer 
configtxgen -profile TwoOrgsOrdererGenesis -configPath $(dirname ${CONFIG_PATH}) -outputBlock ${ARTIFACTS_PATH}/genesis.block -channelID myChannel

# Generate the channel configuration transaction 
configtxgen -profile TwoOrgsChannel -configPath $(dirname ${CONFIG_PATH}) -outputCreateChannelTx ${ARTIFACTS_PATH}/channel.tx -channelID myChannel
