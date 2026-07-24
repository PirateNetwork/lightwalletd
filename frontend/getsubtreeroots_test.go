// Copyright (c) 2026 Pirate Chain developers
// Distributed under the MIT software license, see the accompanying
// file COPYING or https://www.opensource.org/licenses/mit-license.php .

package frontend

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/PirateNetwork/lightwalletd/common"
	"github.com/PirateNetwork/lightwalletd/parser"
	"github.com/PirateNetwork/lightwalletd/walletrpc"
	protobuf "github.com/golang/protobuf/proto"
	"google.golang.org/grpc/metadata"
)

const (
	validSubtreeRoot = "0102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f20"
	validBlockHash   = "00112233445566778899aabbccddeeff00112233445566778899aabbccddeeff"
)

type subtreeRootsStream struct {
	ctx     context.Context
	roots   []*walletrpc.SubtreeRoot
	sendErr error
}

func (s *subtreeRootsStream) SetHeader(metadata.MD) error  { return nil }
func (s *subtreeRootsStream) SendHeader(metadata.MD) error { return nil }
func (s *subtreeRootsStream) SetTrailer(metadata.MD)       {}
func (s *subtreeRootsStream) Context() context.Context     { return s.ctx }
func (s *subtreeRootsStream) SendMsg(interface{}) error    { return nil }
func (s *subtreeRootsStream) RecvMsg(interface{}) error    { return nil }

func (s *subtreeRootsStream) Send(root *walletrpc.SubtreeRoot) error {
	if s.sendErr != nil {
		return s.sendErr
	}
	s.roots = append(s.roots, root)
	return nil
}

func uint64Pointer(value uint64) *uint64 {
	return &value
}

func validRPCSubtree(index, height uint64) common.PiratedRpcReplyGetsubtreesbyindex {
	return common.PiratedRpcReplyGetsubtreesbyindex{
		Index:                 uint64Pointer(index),
		Root:                  validSubtreeRoot,
		CompletingBlockHash:   validBlockHash,
		CompletingBlockHeight: uint64Pointer(height),
	}
}

func subtreeResponse(t *testing.T, roots ...common.PiratedRpcReplyGetsubtreesbyindex) json.RawMessage {
	t.Helper()
	response, err := json.Marshal(roots)
	if err != nil {
		t.Fatal("failed to marshal subtree response:", err)
	}
	return response
}

func installSubtreeRPC(
	t *testing.T,
	expectedProtocol string,
	expectedStartIndex uint32,
	expectedMaxEntries uint32,
	response json.RawMessage,
	rpcErr error,
) {
	t.Helper()
	previous := common.RawRequest
	t.Cleanup(func() {
		common.RawRequest = previous
	})

	common.RawRequest = func(method string, params []json.RawMessage) (json.RawMessage, error) {
		if method != "z_getsubtreesbyindex" {
			t.Fatalf("unexpected RPC method: %s", method)
		}
		if len(params) != 3 {
			t.Fatalf("unexpected parameter count: %d", len(params))
		}

		var protocol string
		if err := json.Unmarshal(params[0], &protocol); err != nil {
			t.Fatal("failed to parse protocol parameter:", err)
		}
		if protocol != expectedProtocol {
			t.Fatalf("unexpected protocol parameter: %s", protocol)
		}

		var startIndex uint32
		if err := json.Unmarshal(params[1], &startIndex); err != nil {
			t.Fatal("failed to parse start index:", err)
		}
		if startIndex != expectedStartIndex {
			t.Fatalf("unexpected start index: %d", startIndex)
		}

		var maxEntries uint32
		if err := json.Unmarshal(params[2], &maxEntries); err != nil {
			t.Fatal("failed to parse maximum entries:", err)
		}
		if maxEntries != expectedMaxEntries {
			t.Fatalf("unexpected maximum entries: %d", maxEntries)
		}

		return response, rpcErr
	}
}

func newSubtreeStreamer(t *testing.T) *lwdStreamer {
	t.Helper()
	streamer, err := NewLwdStreamer(nil, "main", false)
	if err != nil {
		t.Fatal("NewLwdStreamer failed:", err)
	}
	return streamer.(*lwdStreamer)
}

func TestShieldedProtocolUsesOfficialWireValues(t *testing.T) {
	if walletrpc.ShieldedProtocol_sapling != 0 ||
		walletrpc.ShieldedProtocol_orchard != 1 ||
		walletrpc.ShieldedProtocol_ironwood != 2 {
		t.Fatal("shielded protocol wire values do not match the official protocol")
	}

	encoded, err := protobuf.Marshal(&walletrpc.GetSubtreeRootsArg{
		ShieldedProtocol: walletrpc.ShieldedProtocol_ironwood,
	})
	if err != nil {
		t.Fatal("failed to encode Ironwood subtree request:", err)
	}
	if !bytes.Equal(encoded, []byte{0x10, 0x02}) {
		t.Fatalf("unexpected Ironwood subtree request wire bytes: %x", encoded)
	}
}

func TestGetSubtreeRootsBridgesFullNodeResponse(t *testing.T) {
	tests := []struct {
		name     string
		protocol walletrpc.ShieldedProtocol
		rpcName  string
	}{
		{name: "Sapling", protocol: walletrpc.ShieldedProtocol_sapling, rpcName: "sapling"},
		{name: "Ironwood", protocol: walletrpc.ShieldedProtocol_ironwood, rpcName: "ironwood"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			response := subtreeResponse(
				t,
				validRPCSubtree(5, 12345),
				validRPCSubtree(6, 12400),
			)
			installSubtreeRPC(t, tc.rpcName, 5, 2, response, nil)

			stream := &subtreeRootsStream{ctx: context.Background()}
			err := newSubtreeStreamer(t).GetSubtreeRoots(&walletrpc.GetSubtreeRootsArg{
				StartIndex:       5,
				ShieldedProtocol: tc.protocol,
				MaxEntries:       2,
			}, stream)
			if err != nil {
				t.Fatal("GetSubtreeRoots failed:", err)
			}
			if len(stream.roots) != 2 {
				t.Fatalf("unexpected subtree root count: %d", len(stream.roots))
			}

			expectedRoot, _ := hex.DecodeString(validSubtreeRoot)
			if !bytes.Equal(stream.roots[0].RootHash, expectedRoot) {
				t.Fatal("unexpected root hash bytes")
			}
			expectedBlockHash, _ := hex.DecodeString(validBlockHash)
			if !bytes.Equal(
				stream.roots[0].CompletingBlockHash,
				parser.Reverse(expectedBlockHash),
			) {
				t.Fatal("unexpected completing block hash bytes")
			}
			if stream.roots[0].CompletingBlockHeight != 12345 {
				t.Fatal("unexpected completing block height:", stream.roots[0].CompletingBlockHeight)
			}
		})
	}
}

func TestValidateAndConvertSubtreeRootsRejectsMalformedResponses(t *testing.T) {
	missingIndex := validRPCSubtree(5, 100)
	missingIndex.Index = nil
	missingHeight := validRPCSubtree(5, 100)
	missingHeight.CompletingBlockHeight = nil
	invalidRootHex := validRPCSubtree(5, 100)
	invalidRootHex.Root = "not-hex"
	shortRoot := validRPCSubtree(5, 100)
	shortRoot.Root = "00"
	invalidBlockHashHex := validRPCSubtree(5, 100)
	invalidBlockHashHex.CompletingBlockHash = "not-hex"
	shortBlockHash := validRPCSubtree(5, 100)
	shortBlockHash.CompletingBlockHash = "00"

	tests := []struct {
		name       string
		maxEntries uint32
		roots      []common.PiratedRpcReplyGetsubtreesbyindex
		wantError  string
	}{
		{
			name:       "too many entries",
			maxEntries: 1,
			roots:      []common.PiratedRpcReplyGetsubtreesbyindex{validRPCSubtree(5, 100), validRPCSubtree(6, 200)},
			wantError:  "exceeding requested maximum",
		},
		{
			name:      "missing index",
			roots:     []common.PiratedRpcReplyGetsubtreesbyindex{missingIndex},
			wantError: "missing index",
		},
		{
			name:      "wrong first index",
			roots:     []common.PiratedRpcReplyGetsubtreesbyindex{validRPCSubtree(6, 100)},
			wantError: "returned index 6, expected 5",
		},
		{
			name:      "index gap",
			roots:     []common.PiratedRpcReplyGetsubtreesbyindex{validRPCSubtree(5, 100), validRPCSubtree(7, 200)},
			wantError: "returned index 7, expected 6",
		},
		{
			name:      "missing completion height",
			roots:     []common.PiratedRpcReplyGetsubtreesbyindex{missingHeight},
			wantError: "missing completingBlockHeight",
		},
		{
			name:      "equal completion heights",
			roots:     []common.PiratedRpcReplyGetsubtreesbyindex{validRPCSubtree(5, 100), validRPCSubtree(6, 100)},
			wantError: "is not greater than previous height",
		},
		{
			name:      "descending completion heights",
			roots:     []common.PiratedRpcReplyGetsubtreesbyindex{validRPCSubtree(5, 101), validRPCSubtree(6, 100)},
			wantError: "is not greater than previous height",
		},
		{
			name:      "invalid root hex",
			roots:     []common.PiratedRpcReplyGetsubtreesbyindex{invalidRootHex},
			wantError: "invalid subtree root",
		},
		{
			name:      "wrong root length",
			roots:     []common.PiratedRpcReplyGetsubtreesbyindex{shortRoot},
			wantError: "is 1 bytes, expected 32",
		},
		{
			name:      "invalid block hash hex",
			roots:     []common.PiratedRpcReplyGetsubtreesbyindex{invalidBlockHashHex},
			wantError: "invalid completing block hash",
		},
		{
			name:      "wrong block hash length",
			roots:     []common.PiratedRpcReplyGetsubtreesbyindex{shortBlockHash},
			wantError: "is 1 bytes, expected 32",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := validateAndConvertSubtreeRoots(&walletrpc.GetSubtreeRootsArg{
				StartIndex: 5,
				MaxEntries: tc.maxEntries,
			}, tc.roots)
			if err == nil {
				t.Fatal("malformed subtree response was accepted")
			}
			if !strings.Contains(err.Error(), tc.wantError) {
				t.Fatalf("unexpected error %q, expected it to contain %q", err, tc.wantError)
			}
		})
	}
}

func TestValidateAndConvertSubtreeRootsAllowsUnlimitedEntries(t *testing.T) {
	roots, err := validateAndConvertSubtreeRoots(
		&walletrpc.GetSubtreeRootsArg{StartIndex: 5, MaxEntries: 0},
		[]common.PiratedRpcReplyGetsubtreesbyindex{
			validRPCSubtree(5, 100),
			validRPCSubtree(6, 200),
		},
	)
	if err != nil {
		t.Fatal("valid unlimited subtree response was rejected:", err)
	}
	if len(roots) != 2 {
		t.Fatalf("unexpected converted root count: %d", len(roots))
	}
}

func TestGetSubtreeRootsRejectsInvalidJSONShapes(t *testing.T) {
	tests := []struct {
		name      string
		response  json.RawMessage
		wantError string
	}{
		{name: "invalid JSON", response: json.RawMessage("["), wantError: "invalid z_getsubtreesbyindex response"},
		{name: "JSON object", response: json.RawMessage("{}"), wantError: "invalid z_getsubtreesbyindex response"},
		{name: "JSON null", response: json.RawMessage("null"), wantError: "expected an array"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			installSubtreeRPC(t, "sapling", 0, 0, tc.response, nil)
			err := newSubtreeStreamer(t).GetSubtreeRoots(
				&walletrpc.GetSubtreeRootsArg{},
				&subtreeRootsStream{ctx: context.Background()},
			)
			if err == nil || !strings.Contains(err.Error(), tc.wantError) {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestGetSubtreeRootsValidatesBatchBeforeStreaming(t *testing.T) {
	response := subtreeResponse(
		t,
		validRPCSubtree(5, 100),
		validRPCSubtree(7, 200),
	)
	installSubtreeRPC(t, "sapling", 5, 0, response, nil)

	stream := &subtreeRootsStream{ctx: context.Background()}
	err := newSubtreeStreamer(t).GetSubtreeRoots(
		&walletrpc.GetSubtreeRootsArg{StartIndex: 5},
		stream,
	)
	if err == nil {
		t.Fatal("malformed batch was accepted")
	}
	if len(stream.roots) != 0 {
		t.Fatalf("stream received %d roots from an invalid batch", len(stream.roots))
	}
}

func TestGetSubtreeRootsRejectsUnsupportedRequests(t *testing.T) {
	streamer := newSubtreeStreamer(t)
	stream := &subtreeRootsStream{ctx: context.Background()}

	tests := []struct {
		name string
		arg  *walletrpc.GetSubtreeRootsArg
		resp walletrpc.CompactTxStreamer_GetSubtreeRootsServer
	}{
		{name: "missing request", arg: nil, resp: stream},
		{name: "missing stream", arg: &walletrpc.GetSubtreeRootsArg{}, resp: nil},
		{
			name: "Orchard",
			arg: &walletrpc.GetSubtreeRootsArg{
				ShieldedProtocol: walletrpc.ShieldedProtocol_orchard,
			},
			resp: stream,
		},
		{
			name: "unknown protocol",
			arg: &walletrpc.GetSubtreeRootsArg{
				ShieldedProtocol: walletrpc.ShieldedProtocol(99),
			},
			resp: stream,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if err := streamer.GetSubtreeRoots(tc.arg, tc.resp); err == nil {
				t.Fatal("unsupported request was accepted")
			}
		})
	}
}

func TestGetSubtreeRootsPropagatesBoundaryFailures(t *testing.T) {
	t.Run("RPC failure", func(t *testing.T) {
		installSubtreeRPC(t, "sapling", 0, 0, nil, errors.New("node unavailable"))
		err := newSubtreeStreamer(t).GetSubtreeRoots(
			&walletrpc.GetSubtreeRootsArg{},
			&subtreeRootsStream{ctx: context.Background()},
		)
		if err == nil || !strings.Contains(err.Error(), "node unavailable") {
			t.Fatalf("unexpected RPC error: %v", err)
		}
	})

	t.Run("stream failure", func(t *testing.T) {
		installSubtreeRPC(
			t,
			"sapling",
			0,
			0,
			subtreeResponse(t, validRPCSubtree(0, 100)),
			nil,
		)
		err := newSubtreeStreamer(t).GetSubtreeRoots(
			&walletrpc.GetSubtreeRootsArg{},
			&subtreeRootsStream{
				ctx:     context.Background(),
				sendErr: errors.New("stream closed"),
			},
		)
		if err == nil || !strings.Contains(err.Error(), "stream closed") {
			t.Fatalf("unexpected stream error: %v", err)
		}
	})

	t.Run("cancelled request", func(t *testing.T) {
		previous := common.RawRequest
		t.Cleanup(func() {
			common.RawRequest = previous
		})
		rpcCalled := false
		common.RawRequest = func(string, []json.RawMessage) (json.RawMessage, error) {
			rpcCalled = true
			return json.RawMessage("[]"), nil
		}

		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		err := newSubtreeStreamer(t).GetSubtreeRoots(
			&walletrpc.GetSubtreeRootsArg{},
			&subtreeRootsStream{ctx: ctx},
		)
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("unexpected cancellation error: %v", err)
		}
		if rpcCalled {
			t.Fatal("full-node RPC was called after request cancellation")
		}
	})
}
