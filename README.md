# Overview

Zecwallet lightwalletd is a fork of [lightwalletd](https://github.com/adityapk00/lightwalletd) from the ECC. 

It is a backend service that provides a bandwidth-efficient interface to the Zcash blockchain for the [Zecwallet light wallet](https://github.com/adityapk00/zecwallet-lite-lib).

## Changes from upstream lightwalletd
This version of Zecwallet lightwalletd extends lightwalletd and:
* Adds support for transparent addresses
* Adds several new RPC calls for lightclients
* Lots of perf improvements
  * Replaces SQLite with in-memory cache for Compact Blocks
  * Replace local Txstore, delegating Tx lookups to Zcashd
  * Remove the need for a separate ingestor

## Running your own zeclite lightwalletd

#### 0. First, install [Go >= 1.11](https://golang.org/dl/#stable).

#### 1. Run a zcash node.
Start a `zcashd` with the following options:
```
server=1
rpcuser=user
rpcpassword=password
rpcbind=127.0.0.1
rpcport=8232
experimentalfeatures=1
txindex=1
insightexplorer=1
```

You might need to run with `-reindex` the first time if you are enabling the `txindex` or `insightexplorer` options for the first time. The reindex might take a while.

#### 2. Get a TLS certificate
##### a. Self-signed certificate if you are running a private instance just for yourself:

Generate a TLS self-signed certificate
```
openssl req -x509 -newkey rsa:4096 -keyout key.pem -out cert.pem -days 365 -nodes
```
Answer the certificate questions to generate the self-signed certificate
##### b. "Let's Encrypt" certificate using NGINX as a reverse proxy
If you running a public-facing server, the easiest way to obtain a certificate is to use a NGINX reverse proxy and get a Let's Encrypt certificate. [Instructions are here](https://www.nginx.com/blog/using-free-ssltls-certificates-from-lets-encrypt-with-nginx/)

Create a new section for the NGINX reverse proxy:
```
server {
    listen 443 ssl http2;
 
 
    ssl_certificate     ssl/cert.pem; # From certbot
    ssl_certificate_key ssl/key.pem;  # From certbot
    
    location / {
        # Replace localhost:9067 with the address and port of your gRPC server if using a custom port
        grpc_pass grpc://localhost:9067;
    }
}
```

##### c. Use your own TLS certificate
If you have your own TLS certificate issued by a certificate authority, you can use it directly with the GRPC frontend, as described in the next section

#### 3. Run the frontend:
You can run the gRPC server with or without TLS, depending on how you configured step 2. If you are using NGINX as a reverse proxy and are letting NGINX handle the TLS authentication, then run the frontend with `-no-tls`

```
go run cmd/server/main.go -bind-addr 127.0.0.1:9067 -conf-file ~/.zcash/zcash.conf -no-tls
```

If you have a certificate that you want to use (either self signed, or from a certificate authority), pass the certificate to the frontend:

```
go run cmd/server/main.go -bind-addr 127.0.0.1:443 -conf-file ~/.zcash/zcash.conf  -tls-cert cert.pem -tls-key key.pem
```

You should start seeing the frontend ingest and cache the zcash blocks after ~15 seconds. 

#### 4. Point the `zecwallet-cli` to this server
Connect to your server!
```
./zecwallet-cli -server https://mylightwalletd.server.com
```

If you are using a self-signed certificate, then zecwallet will reject the connection by default (because the certificate isn't from a valid certificate authority. To override this, pass the `--dangerous` flag. Obviously, this is dangerous, so please only use if you are connecting to **your own** server.

```
./zecwallet-cli --server https://127.0.0.1:9067 --dangerous
```
