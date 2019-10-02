# Overview

Zecwallet lightwalletd is a fork of [lightwalletd](https://github.com/adityapk00/lightwalletd) from the ECC. 

It is a backend service that provides a bandwidth-efficient interface to the Zcash blockchain for the [Zecwallet light wallet](https://github.com/adityapk00/lightwalletclient).

## Changes from upstream lightwalletd
This version of zeclite lightwalletd extends lightwalletd and:
* Adds support for transparent addresses
* Adds several new RPC calls for lightclients
* Lots of perf improvements
  * Replaces SQLite with in-memory cache for Compact Blocks
  * Replace local Txstore, delegating Tx lookups to Zcashd
  * Remove the need for a separate ingestor

## Running your own zeclite lightwalletd

0. First, install [Go >= 1.11](https://golang.org/dl/#stable).

1. You need to run a zcash full node with the following options in zcash.conf
```
server=1
testnet=1
rpcuser=user
rpcpassword=password
rpcbind=127.0.0.1
rpcport=18232
experimentalfeatures=1
txindex=1
insightexplorer=1
```

2. Run the frontend:
```
go run cmd/server/main.go -bind-addr 127.0.0.1:9067 -conf-file ~/.zcash/zcash.conf
```

3. Point the `zecwallet-cli` to this server
```
./zeclite-cli --server 127.0.0.1:9067
```
