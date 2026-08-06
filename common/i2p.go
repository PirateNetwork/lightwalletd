// Copyright (c) 2026 The Pirate Chain developers
// Distributed under the MIT software license, see the accompanying
// file COPYING or https://www.opensource.org/licenses/mit-license.php .

package common

import (
	"io"
	"net"
	"os"
	"path/filepath"
	"time"

	"github.com/go-i2p/sam3"
	"github.com/sirupsen/logrus"
)

// StartI2PServer publishes lightwalletd's gRPC port as an I2P destination,
// reusing the i2pd daemon that TreasureChest already starts and manages
// (i2pdautostart/-i2psam in PIRATE.conf) rather than launching an I2P
// router of its own. It runs in the background: failures are logged as
// warnings and retried with backoff, and never prevent lightwalletd from
// serving clearnet clients.
//
// Unlike Tor's ADD_ONION, I2P's SAM API has no "forward inbound streams to
// this local TCP port" primitive, so lightwalletd accepts each inbound I2P
// stream itself and proxies it to its own local gRPC listener.
func StartI2PServer(opts *Options, grpcAddr string) {
	go runI2PServer(opts, grpcAddr)
}

func runI2PServer(opts *Options, grpcAddr string) {
	const minBackoff = 5 * time.Second
	const maxBackoff = 2 * time.Minute
	backoff := minBackoff
	// TreasureChest's embedded i2pd picks a different SAM port at runtime if
	// -i2psam's configured port was already taken, and that override never
	// makes it back into PIRATE.conf - so what we start with (flag/conf
	// default) can already be wrong. Track it as a mutable local rather than
	// re-reading opts.I2PSamAddr, and correct it via RPC below whenever a
	// connection attempt fails.
	samAddr := opts.I2PSamAddr
	for {
		err := publishI2PDestination(opts, samAddr, grpcAddr)
		if err != nil {
			Log.WithFields(logrus.Fields{
				"error":    err,
				"sam_addr": samAddr,
			}).Warn("i2p: destination unavailable, will retry")
			if _, actual, rpcErr := GetTorI2PInfoFromRPC(); rpcErr == nil && actual != "" && actual != samAddr {
				Log.WithFields(logrus.Fields{
					"configured_addr": samAddr,
					"actual_addr":     actual,
				}).Info("i2p: pirated reports a different SAM address than configured, switching to it")
				samAddr = actual
			}
			time.Sleep(backoff)
			if backoff < maxBackoff {
				backoff *= 2
			}
			continue
		}
		backoff = minBackoff
	}
}

// publishI2PDestination connects to the I2P SAM API, creates (or loads a
// persisted) destination, and accepts inbound streams until the session
// fails, proxying each one to the local gRPC listener. Returns when the
// session drops, so the caller can reconnect.
func publishI2PDestination(opts *Options, samAddr, grpcAddr string) error {
	sam, err := sam3.NewSAM(samAddr)
	if err != nil {
		return err
	}
	defer sam.Close()

	// sam3's EnsureKeyfile opens the keyfile with O_CREATE but never
	// creates its parent directory, so on a fresh data-dir it fails
	// outright the first time; create it ourselves first (matching
	// saveOnionKeyIfNew's behavior on the Tor side).
	if opts.I2PKeysFile != "" {
		if err := os.MkdirAll(filepath.Dir(opts.I2PKeysFile), 0700); err != nil {
			return err
		}
	}
	keys, err := sam.EnsureKeyfile(opts.I2PKeysFile)
	if err != nil {
		return err
	}

	session, err := sam.NewStreamSession("lightwalletd", keys, sam3.Options_Default)
	if err != nil {
		return err
	}
	defer session.Close()

	listener, err := session.Listen()
	if err != nil {
		return err
	}
	defer listener.Close()

	address := keys.Addr().Base32()
	writeHostnameFile(opts.I2PKeysFile, address)

	Log.WithFields(logrus.Fields{
		"address": address,
	}).Info("i2p: destination published")

	for {
		conn, err := listener.Accept()
		if err != nil {
			return err
		}
		go proxyI2PConn(conn, grpcAddr)
	}
}

func proxyI2PConn(i2pConn net.Conn, grpcAddr string) {
	defer i2pConn.Close()

	localConn, err := net.DialTimeout("tcp", grpcAddr, 10*time.Second)
	if err != nil {
		Log.WithField("error", err).Warn("i2p: couldn't reach local gRPC listener for inbound stream")
		return
	}
	defer localConn.Close()

	done := make(chan struct{}, 2)
	go func() {
		io.Copy(localConn, i2pConn)
		done <- struct{}{}
	}()
	go func() {
		io.Copy(i2pConn, localConn)
		done <- struct{}{}
	}()
	<-done
}
