// Copyright (c) 2026 The Pirate Chain developers
// Distributed under the MIT software license, see the accompanying
// file COPYING or https://www.opensource.org/licenses/mit-license.php .

package common

import (
	"net"
	"net/textproto"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/cretz/bine/control"
	"github.com/sirupsen/logrus"
)

// StartTorHiddenService publishes lightwalletd's gRPC and HTTP ports as a
// Tor hidden service, reusing the Tor daemon that TreasureChest already
// starts and manages (torautostart/-torcontrol in PIRATE.conf) rather than
// launching a Tor process of its own. It runs in the background: failures
// are logged as warnings and retried with backoff, and never prevent
// lightwalletd from serving clearnet clients.
func StartTorHiddenService(opts *Options, grpcAddr, httpAddr string) {
	grpcPort, err := portOf(grpcAddr)
	if err != nil {
		Log.WithField("error", err).Error("tor: invalid grpc-bind-addr, not starting hidden service")
		return
	}
	httpPort, err := portOf(httpAddr)
	if err != nil {
		Log.WithField("error", err).Error("tor: invalid http-bind-addr, not starting hidden service")
		return
	}
	go runTorHiddenService(opts, grpcPort, httpPort)
}

func runTorHiddenService(opts *Options, grpcPort, httpPort int) {
	const minBackoff = 5 * time.Second
	const maxBackoff = 2 * time.Minute
	backoff := minBackoff
	// TreasureChest's embedded tor daemon picks a different control port at
	// runtime if -torcontrol's configured port was already taken, and that
	// override never makes it back into PIRATE.conf - so what we start with
	// (flag/conf default) can already be wrong. Track it as a mutable local
	// rather than re-reading opts.TorControlAddr, and correct it via RPC
	// below whenever a connection attempt fails.
	controlAddr := opts.TorControlAddr
	for {
		err := publishOnion(opts, controlAddr, grpcPort, httpPort)
		if err != nil {
			Log.WithFields(logrus.Fields{
				"error":        err,
				"control_addr": controlAddr,
			}).Warn("tor: hidden service unavailable, will retry")
			if actual, _, rpcErr := GetTorI2PInfoFromRPC(); rpcErr == nil && actual != "" && actual != controlAddr {
				Log.WithFields(logrus.Fields{
					"configured_addr": controlAddr,
					"actual_addr":     actual,
				}).Info("tor: pirated reports a different control port than configured, switching to it")
				controlAddr = actual
			}
		}
		time.Sleep(backoff)
		if backoff < maxBackoff {
			backoff *= 2
		}
		if err == nil {
			// We had a working session that later dropped; reset backoff
			// so a transient blip doesn't leave us waiting minutes to retry.
			backoff = minBackoff
		}
	}
}

// publishOnion connects to the Tor control port, authenticates, adds the
// onion service, and then blocks (issuing periodic no-op commands to detect
// a dead connection) until the control connection is lost - at which point
// the hidden service also disappears (we don't pass the Detach flag, to
// match the reconnect-and-republish behavior of TreasureChest's own
// torcontrol.cpp) and the caller should retry.
func publishOnion(opts *Options, controlAddr string, grpcPort, httpPort int) error {
	conn, err := net.DialTimeout("tcp", controlAddr, 10*time.Second)
	if err != nil {
		return err
	}
	defer conn.Close()

	c := control.NewConn(textproto.NewConn(conn))
	if err := c.Authenticate(opts.TorPassword); err != nil {
		return err
	}

	key, err := loadOnionKey(opts.TorKeysFile)
	if err != nil {
		return err
	}

	resp, err := c.AddOnion(&control.AddOnionRequest{
		Key: key,
		Ports: []*control.KeyVal{
			{Key: strconv.Itoa(grpcPort), Val: "127.0.0.1:" + strconv.Itoa(grpcPort)},
			{Key: strconv.Itoa(httpPort), Val: "127.0.0.1:" + strconv.Itoa(httpPort)},
		},
	})
	if err != nil {
		return err
	}
	defer c.DelOnion(resp.ServiceID)

	if err := saveOnionKeyIfNew(opts.TorKeysFile, resp.Key); err != nil {
		Log.WithField("error", err).Warn("tor: couldn't persist onion key, address will change on restart")
	}
	address := resp.ServiceID + ".onion"
	writeHostnameFile(opts.TorKeysFile, address)

	Log.WithFields(logrus.Fields{
		"address":   address,
		"grpc_port": grpcPort,
		"http_port": httpPort,
	}).Info("tor: hidden service published")

	// Keep the control connection open (dropping it removes the onion
	// service); a periodic no-op both keeps it alive and detects a dead
	// connection promptly instead of waiting for a write to fail.
	for {
		time.Sleep(60 * time.Second)
		if _, err := c.SendRequest("GETINFO version"); err != nil {
			return err
		}
	}
}

func loadOnionKey(path string) (control.Key, error) {
	if path == "" {
		return control.GenKey(control.KeyAlgoBest), nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return control.GenKey(control.KeyAlgoBest), nil
		}
		return nil, err
	}
	return control.KeyFromString(strings.TrimSpace(string(data)))
}

func saveOnionKeyIfNew(path string, key control.Key) error {
	if path == "" || key == nil {
		return nil
	}
	if _, err := os.Stat(path); err == nil {
		return nil // already persisted
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	blob := string(key.Type()) + ":" + key.Blob()
	return os.WriteFile(path, []byte(blob), 0600)
}

// writeHostnameFile records address in a "hostname" file next to keyFile,
// mirroring the convention Tor's own HiddenServiceDir uses, so an operator
// (or tooling) can find the current onion/I2P address with a plain `cat`
// instead of grepping logs. keyFile == "" (ephemeral key, nothing persisted)
// means there's no stable directory to write into, so this is a no-op.
// Rewritten on every publish, not just the first, so it also self-heals if
// deleted or if the address ever changes.
func writeHostnameFile(keyFile, address string) {
	if keyFile == "" {
		return
	}
	dir := filepath.Dir(keyFile)
	if err := os.MkdirAll(dir, 0700); err != nil {
		Log.WithField("error", err).Warn("couldn't create directory for hostname file")
		return
	}
	if err := os.WriteFile(filepath.Join(dir, "hostname"), []byte(address+"\n"), 0644); err != nil {
		Log.WithField("error", err).Warn("couldn't write hostname file")
	}
}

func portOf(addr string) (int, error) {
	_, portStr, err := net.SplitHostPort(addr)
	if err != nil {
		return 0, err
	}
	return strconv.Atoi(portStr)
}
