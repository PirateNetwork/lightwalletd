#!/bin/bash

CGO_ENABLED=0 go build -a -ldflags '-extldflags "-static"' cmd/server/main.go 
docker build --tag lightwalletd:latest -f docker/Dockerfile .
