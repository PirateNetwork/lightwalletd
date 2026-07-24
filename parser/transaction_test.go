// Copyright (c) 2019-2020 The Zcash developers
// Copyright (c) 2019-2021 Pirate Chain developers
// Distributed under the MIT software license, see the accompanying
// file COPYING or https://www.opensource.org/licenses/mit-license.php .
package parser

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"os"
	"strings"
	"testing"
)

// Some of these values may be "null" (which translates to nil in Go) in
// the test data, so we have *_set variables to indicate if the corresponding
// variable is non-null. (There is an "optional" package we could use for
// these but it doesn't seem worth pulling it in.)
type TxTestData struct {
	Tx                 string
	Txid               string
	Version            int
	NVersionGroupId    int
	NConsensusBranchId int
	Tx_in_count        int
	Tx_out_count       int
	NSpendsSapling     int
	NoutputsSapling    int
	NActionsOrchard    int
}

// https://jhall.io/posts/go-json-tricks-array-as-structs/
func (r *TxTestData) UnmarshalJSON(p []byte) error {
	var t []interface{}
	if err := json.Unmarshal(p, &t); err != nil {
		return err
	}
	r.Tx = t[0].(string)
	r.Txid = t[1].(string)
	r.Version = int(t[2].(float64))
	r.NVersionGroupId = int(t[3].(float64))
	r.NConsensusBranchId = int(t[4].(float64))
	r.Tx_in_count = int(t[7].(float64))
	r.Tx_out_count = int(t[8].(float64))
	r.NSpendsSapling = int(t[9].(float64))
	r.NoutputsSapling = int(t[10].(float64))
	r.NActionsOrchard = int(t[14].(float64))
	return nil
}

func TestV5TransactionParser(t *testing.T) {
	// The raw data are stored in a separate file because they're large enough
	// to make the test table difficult to scroll through. They are in the same
	// order as the test table above. If you update the test table without
	// adding a line to the raw file, this test will panic due to index
	// misalignment.
	s, err := os.ReadFile("../testdata/tx_v5.json")
	if err != nil {
		t.Fatal(err)
	}

	var testdata []json.RawMessage
	err = json.Unmarshal(s, &testdata)
	if err != nil {
		t.Fatal(err)
	}
	if len(testdata) < 3 {
		t.Fatal("tx_vt.json has too few lines")
	}
	testdata = testdata[2:]
	for _, onetx := range testdata {
		var txtestdata TxTestData

		err = json.Unmarshal(onetx, &txtestdata)
		if err != nil {
			t.Fatal(err)
		}
		t.Logf("txid %s", txtestdata.Txid)
		rawTxData, _ := hex.DecodeString(txtestdata.Tx)

		tx := NewTransaction()
		rest, err := tx.ParseFromSlice(rawTxData)
		if err != nil {
			t.Fatalf("%v", err)
		}
		if len(rest) != 0 {
			t.Fatalf("Test did not consume entire buffer, %d remaining", len(rest))
		}
		// Currently, we can't check the txid because we get that from
		// zcashd (getblock rpc) rather than computing it ourselves.
		// https://github.com/zcash/lightwalletd/issues/392
		if tx.version != uint32(txtestdata.Version) {
			t.Fatal("version miscompare")
		}
		if tx.nVersionGroupID != uint32(txtestdata.NVersionGroupId) {
			t.Fatal("nVersionGroupId miscompare")
		}
		if tx.consensusBranchID != uint32(txtestdata.NConsensusBranchId) {
			t.Fatal("consensusBranchID miscompare")
		}
		if len(tx.transparentInputs) != int(txtestdata.Tx_in_count) {
			t.Fatal("tx_in_count miscompare")
		}
		if len(tx.transparentOutputs) != int(txtestdata.Tx_out_count) {
			t.Fatal("tx_out_count miscompare")
		}
		if len(tx.shieldedSpends) != int(txtestdata.NSpendsSapling) {
			t.Fatal("NSpendsSapling miscompare")
		}
		if len(tx.shieldedOutputs) != int(txtestdata.NoutputsSapling) {
			t.Fatal("NOutputsSapling miscompare")
		}
		if len(tx.orchardActions) != int(txtestdata.NActionsOrchard) {
			t.Fatal("NActionsOrchard miscompare")
		}
	}
}

func writeUint32(buf *bytes.Buffer, value uint32) {
	var encoded [4]byte
	binary.LittleEndian.PutUint32(encoded[:], value)
	buf.Write(encoded[:])
}

func writeActionBundle(buf *bytes.Buffer, marker byte) {
	buf.WriteByte(1)
	buf.Write(bytes.Repeat([]byte{marker}, 32))     // cv
	buf.Write(bytes.Repeat([]byte{marker + 1}, 32)) // nullifier
	buf.Write(bytes.Repeat([]byte{marker + 2}, 32)) // rk
	buf.Write(bytes.Repeat([]byte{marker + 3}, 32)) // cmx
	buf.Write(bytes.Repeat([]byte{marker + 4}, 32)) // ephemeral key
	buf.Write(bytes.Repeat([]byte{marker + 5}, 580))
	buf.Write(bytes.Repeat([]byte{marker + 6}, 80))
	buf.WriteByte(1) // flags
	buf.Write(make([]byte, 8))
	buf.Write(make([]byte, 32))
	buf.WriteByte(0) // proof length
	buf.Write(make([]byte, 64))
	buf.Write(make([]byte, 64))
}

func buildV6Transaction(groupID uint32, includeOrchard bool) []byte {
	var buf bytes.Buffer
	writeUint32(&buf, uint32(1<<31)|ironwoodTxVersion)
	writeUint32(&buf, groupID)
	writeUint32(&buf, 0x37A5165B)
	writeUint32(&buf, 0)
	writeUint32(&buf, 0)
	buf.WriteByte(0) // transparent inputs
	buf.WriteByte(0) // transparent outputs
	buf.WriteByte(0) // Sapling spends
	buf.WriteByte(0) // Sapling outputs
	if includeOrchard {
		writeActionBundle(&buf, 0x20)
	} else {
		buf.WriteByte(0)
	}
	writeActionBundle(&buf, 0x40)
	return buf.Bytes()
}

func TestV6IronwoodTransactionParser(t *testing.T) {
	raw := buildV6Transaction(ironwoodVersionGroupID, false)
	tx := NewTransaction()

	rest, err := tx.ParseFromSlice(raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(rest) != 0 {
		t.Fatalf("parser left %d transaction bytes unconsumed", len(rest))
	}
	if tx.version != ironwoodTxVersion {
		t.Fatalf("unexpected transaction version: %d", tx.version)
	}
	if tx.nVersionGroupID != ironwoodVersionGroupID {
		t.Fatalf("unexpected version group ID: 0x%08X", tx.nVersionGroupID)
	}
	if tx.consensusBranchID != 0x37A5165B {
		t.Fatalf("unexpected consensus branch ID: 0x%08X", tx.consensusBranchID)
	}
	if len(tx.orchardActions) != 0 {
		t.Fatalf("unexpected Orchard action count: %d", len(tx.orchardActions))
	}
	if len(tx.ironwoodActions) != 1 {
		t.Fatalf("unexpected Ironwood action count: %d", len(tx.ironwoodActions))
	}
	if !tx.HasShieldedElements() {
		t.Fatal("Ironwood transaction was not recognized as shielded")
	}

	compact := tx.ToCompact(7)
	if len(compact.Actions) != 1 {
		t.Fatalf("unexpected compact action count: %d", len(compact.Actions))
	}
	action := compact.Actions[0]
	if !bytes.Equal(action.Nullifier, bytes.Repeat([]byte{0x41}, 32)) {
		t.Fatal("unexpected compact Ironwood nullifier")
	}
	if !bytes.Equal(action.Cmx, bytes.Repeat([]byte{0x43}, 32)) {
		t.Fatal("unexpected compact Ironwood commitment")
	}
	if !bytes.Equal(action.EphemeralKey, bytes.Repeat([]byte{0x44}, 32)) {
		t.Fatal("unexpected compact Ironwood ephemeral key")
	}
	if !bytes.Equal(action.Ciphertext, bytes.Repeat([]byte{0x45}, 52)) {
		t.Fatal("unexpected compact Ironwood ciphertext")
	}
}

func TestV6TransactionRejectsWrongGroupID(t *testing.T) {
	tx := NewTransaction()
	_, err := tx.ParseFromSlice(buildV6Transaction(zip225VersionGroupID, false))
	if err == nil || !strings.Contains(err.Error(), "version group ID") {
		t.Fatalf("expected version group ID error, got %v", err)
	}
}

func TestV6TransactionRejectsOrchardActions(t *testing.T) {
	tx := NewTransaction()
	_, err := tx.ParseFromSlice(buildV6Transaction(ironwoodVersionGroupID, true))
	if err == nil || !strings.Contains(err.Error(), "Orchard pool slot must be empty") {
		t.Fatalf("expected nonempty Orchard slot error, got %v", err)
	}
}
