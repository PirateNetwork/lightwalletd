// Copyright (c) 2026 The Pirate Chain developers
// Distributed under the MIT software license, see the accompanying
// file COPYING or https://www.opensource.org/licenses/mit-license.php .

package common

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/cretz/bine/control"
)

func TestPortOf(t *testing.T) {
	port, err := portOf("127.0.0.1:9067")
	if err != nil {
		t.Fatal("portOf failed on valid address")
	}
	if port != 9067 {
		t.Fatal("portOf returned wrong port")
	}

	if _, err := portOf("not-an-address"); err == nil {
		t.Fatal("portOf unexpected success on invalid address")
	}
}

func TestOnionKeyPersistence(t *testing.T) {
	dir := t.TempDir()
	keyFile := filepath.Join(dir, "tor", "onion_private_key")

	// no file yet, and no path at all: both should hand back a fresh
	// GenKey rather than erroring
	if key, err := loadOnionKey(""); err != nil || key.Type() != control.KeyTypeNew {
		t.Fatal("loadOnionKey with empty path should return a GenKey")
	}
	key, err := loadOnionKey(keyFile)
	if err != nil || key.Type() != control.KeyTypeNew {
		t.Fatal("loadOnionKey with missing file should return a GenKey")
	}

	// saving with an empty path or nil key is a no-op, not an error
	if err := saveOnionKeyIfNew("", key); err != nil {
		t.Fatal("saveOnionKeyIfNew with empty path should be a no-op")
	}
	if _, err := os.Stat(keyFile); !os.IsNotExist(err) {
		t.Fatal("saveOnionKeyIfNew with empty path should not create a file")
	}

	// simulate what Tor's ADD_ONION response would hand back for a
	// generated ED25519-V3 key, and persist it
	generated, err := control.KeyFromString("ED25519-V3:c2FtcGxlLWtleS1ibG9iLWZvci10ZXN0aW5n")
	if err != nil {
		t.Fatal("control.KeyFromString failed on test fixture")
	}
	if err := saveOnionKeyIfNew(keyFile, generated); err != nil {
		t.Fatal("saveOnionKeyIfNew failed")
	}
	if _, err := os.Stat(keyFile); err != nil {
		t.Fatal("saveOnionKeyIfNew did not create the key file")
	}

	// loading it back should reproduce the same key
	loaded, err := loadOnionKey(keyFile)
	if err != nil {
		t.Fatal("loadOnionKey failed to read persisted key")
	}
	if loaded.Type() != generated.Type() || loaded.Blob() != generated.Blob() {
		t.Fatal("loadOnionKey did not round-trip the persisted key")
	}

	// once a key is on disk, saving again must not overwrite it
	other, _ := control.KeyFromString("ED25519-V3:b3RoZXItZ2VuZXJhdGVkLWtleS1ibG9i")
	if err := saveOnionKeyIfNew(keyFile, other); err != nil {
		t.Fatal("saveOnionKeyIfNew (existing file) failed")
	}
	reloaded, err := loadOnionKey(keyFile)
	if err != nil {
		t.Fatal("loadOnionKey failed after no-op save")
	}
	if reloaded.Blob() != generated.Blob() {
		t.Fatal("saveOnionKeyIfNew overwrote an existing persisted key")
	}
}

func TestWriteHostnameFile(t *testing.T) {
	dir := t.TempDir()
	keyFile := filepath.Join(dir, "tor", "onion_private_key")

	// no keyFile path: nothing to write into, must not create anything
	writeHostnameFile("", "should-not-be-written.onion")
	if entries, _ := os.ReadDir(dir); len(entries) != 0 {
		t.Fatal("writeHostnameFile with empty keyFile should not create anything")
	}

	writeHostnameFile(keyFile, "abc123.onion")
	hostnamePath := filepath.Join(dir, "tor", "hostname")
	data, err := os.ReadFile(hostnamePath)
	if err != nil {
		t.Fatal("writeHostnameFile did not create the hostname file")
	}
	if string(data) != "abc123.onion\n" {
		t.Fatal("writeHostnameFile wrote unexpected content", string(data))
	}

	// republishing (e.g. address changed, or self-healing after deletion)
	// must overwrite, not append or error
	writeHostnameFile(keyFile, "xyz789.onion")
	data, err = os.ReadFile(hostnamePath)
	if err != nil {
		t.Fatal("writeHostnameFile (rewrite) failed to read back")
	}
	if string(data) != "xyz789.onion\n" {
		t.Fatal("writeHostnameFile did not overwrite stale content", string(data))
	}
}

func TestGetTorI2PInfoFromRPC(t *testing.T) {
	origRawRequest := RawRequest
	defer func() { RawRequest = origRawRequest }()

	RawRequest = func(method string, params []json.RawMessage) (json.RawMessage, error) {
		if method != "getnetworkinfo" {
			t.Fatalf("unexpected RPC method %q", method)
		}
		return json.Marshal(&PiratedRpcReplyGetnetworkinfo{
			TorControl: "127.0.0.1:9151",
			Networks: []struct {
				Name  string `json:"name"`
				Proxy string `json:"proxy"`
			}{
				{Name: "ipv4", Proxy: ""},
				{Name: "onion", Proxy: "127.0.0.1:9050"},
				{Name: "i2p", Proxy: "127.0.0.1:7756"},
			},
		})
	}
	torControlAddr, i2pSamAddr, err := GetTorI2PInfoFromRPC()
	if err != nil {
		t.Fatal("GetTorI2PInfoFromRPC failed")
	}
	if torControlAddr != "127.0.0.1:9151" {
		t.Fatal("GetTorI2PInfoFromRPC returned unexpected torControlAddr", torControlAddr)
	}
	if i2pSamAddr != "127.0.0.1:7756" {
		t.Fatal("GetTorI2PInfoFromRPC returned unexpected i2pSamAddr", i2pSamAddr)
	}

	// no i2p entry present: should come back empty, not error
	RawRequest = func(method string, params []json.RawMessage) (json.RawMessage, error) {
		return json.Marshal(&PiratedRpcReplyGetnetworkinfo{TorControl: "127.0.0.1:9051"})
	}
	torControlAddr, i2pSamAddr, err = GetTorI2PInfoFromRPC()
	if err != nil {
		t.Fatal("GetTorI2PInfoFromRPC failed")
	}
	if torControlAddr != "127.0.0.1:9051" || i2pSamAddr != "" {
		t.Fatal("GetTorI2PInfoFromRPC returned unexpected values", torControlAddr, i2pSamAddr)
	}

	// RPC failure should propagate, not panic or swallow the error
	RawRequest = func(method string, params []json.RawMessage) (json.RawMessage, error) {
		return nil, errors.New("rpc unavailable")
	}
	if _, _, err := GetTorI2PInfoFromRPC(); err == nil {
		t.Fatal("GetTorI2PInfoFromRPC unexpected success on RPC error")
	}
}
