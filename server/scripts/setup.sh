#!/bin/bash

echo "Setting up Baseball Grid Game on Raspberry Pi..."

# Create project structure
mkdir -p cmd/server
mkdir -p internal/{handlers,models,database,websocket,auth}
mkdir -p pkg/{utils,config}
mkdir -p migrations
mkdir -p logs
mkdir -p static