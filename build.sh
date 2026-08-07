#!/bin/bash
set -e

CGO_ENABLED=0 go build -a -ldflags '-extldflags "-static"' -o lightwalletd .
docker build --tag lightwalletd:latest -f Dockerfile .
