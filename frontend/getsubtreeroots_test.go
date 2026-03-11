// Copyright (c) 2026 Pirate Chain developers
// Distributed under the MIT software license, see the accompanying
// file COPYING or https://www.opensource.org/licenses/mit-license.php .

package frontend

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"testing"

	"github.com/PirateNetwork/lightwalletd/common"
	"github.com/PirateNetwork/lightwalletd/parser"
	"github.com/PirateNetwork/lightwalletd/walletrpc"
	"google.golang.org/grpc/metadata"
)

type subtreeRootsStream struct {
	ctx   context.Context
	roots []*walletrpc.SubtreeRoot
}

func (s *subtreeRootsStream) SetHeader(metadata.MD) error  { return nil }
func (s *subtreeRootsStream) SendHeader(metadata.MD) error { return nil }
func (s *subtreeRootsStream) SetTrailer(metadata.MD)       {}
func (s *subtreeRootsStream) Context() context.Context     { return s.ctx }
func (s *subtreeRootsStream) SendMsg(interface{}) error    { return nil }
func (s *subtreeRootsStream) RecvMsg(interface{}) error    { return nil }

func (s *subtreeRootsStream) Send(root *walletrpc.SubtreeRoot) error {
	s.roots = append(s.roots, root)
	return nil
}

func z_getsubtreesbyindexStub(method string, params []json.RawMessage) (json.RawMessage, error) {
	if method != "z_getsubtreesbyindex" {
		testT.Fatal("unexpected method in z_getsubtreesbyindexStub:", method)
	}
	if len(params) != 3 {
		testT.Fatalf("unexpected params len in z_getsubtreesbyindexStub: %d", len(params))
	}

	var protocol string
	if err := json.Unmarshal(params[0], &protocol); err != nil {
		testT.Fatal("failed to parse protocol param:", err)
	}
	if protocol != "sapling" {
		testT.Fatal("unexpected protocol param:", protocol)
	}

	var startIndex uint32
	if err := json.Unmarshal(params[1], &startIndex); err != nil {
		testT.Fatal("failed to parse startIndex param:", err)
	}
	if startIndex != 5 {
		testT.Fatal("unexpected startIndex param:", startIndex)
	}

	var maxEntries uint32
	if err := json.Unmarshal(params[2], &maxEntries); err != nil {
		testT.Fatal("failed to parse maxEntries param:", err)
	}
	if maxEntries != 2 {
		testT.Fatal("unexpected maxEntries param:", maxEntries)
	}

	mockResponse := `[
		{
			"index": 5,
			"root": "0102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f20",
			"completingBlockHash": "00112233445566778899aabbccddeeff00112233445566778899aabbccddeeff",
			"completingBlockHeight": 12345
		}
	]`

	return json.RawMessage(mockResponse), nil
}

func TestGetSubtreeRoots(t *testing.T) {
	testT = t
	common.RawRequest = z_getsubtreesbyindexStub

	lwdInterface, err := NewLwdStreamer(nil, "main", false)
	if err != nil {
		t.Fatal("NewLwdStreamer failed:", err)
	}
	lwd := lwdInterface.(*lwdStreamer)

	stream := &subtreeRootsStream{ctx: context.Background()}
	err = lwd.GetSubtreeRoots(&walletrpc.GetSubtreeRootsArg{
		StartIndex:       5,
		ShieldedProtocol: walletrpc.ShieldedProtocol_sapling,
		MaxEntries:       2,
	}, stream)
	if err != nil {
		t.Fatal("GetSubtreeRoots failed:", err)
	}

	if len(stream.roots) != 1 {
		t.Fatalf("unexpected subtree root count: %d", len(stream.roots))
	}

	expectedRoot, err := hex.DecodeString("0102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f20")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(stream.roots[0].RootHash, expectedRoot) {
		t.Fatal("unexpected root hash bytes")
	}

	expectedBlockHash, err := hex.DecodeString("00112233445566778899aabbccddeeff00112233445566778899aabbccddeeff")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(stream.roots[0].CompletingBlockHash, parser.Reverse(expectedBlockHash)) {
		t.Fatal("unexpected completing block hash bytes")
	}

	if stream.roots[0].CompletingBlockHeight != 12345 {
		t.Fatal("unexpected completing block height:", stream.roots[0].CompletingBlockHeight)
	}
}
