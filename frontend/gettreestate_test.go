// Copyright (c) 2019-2020 The Zcash developers
// Copyright (c) 2019-2021 Pirate Chain developers
// Distributed under the MIT software license, see the accompanying
// file COPYING or https://www.opensource.org/licenses/mit-license.php .

package frontend

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/PirateNetwork/lightwalletd/common"
	"github.com/PirateNetwork/lightwalletd/walletrpc"
)

func newTreeStateTestStreamer(t *testing.T) *lwdStreamer {
	t.Helper()
	lwdInterface, err := NewLwdStreamer(nil, "main", false)
	if err != nil {
		t.Fatal("NewLwdStreamer failed:", err)
	}
	lwd, ok := lwdInterface.(*lwdStreamer)
	if !ok {
		t.Fatal("NewLwdStreamer did not return *lwdStreamer")
	}
	return lwd
}

func z_gettreestatelegacyStub(method string, params []json.RawMessage) (json.RawMessage, error) {
	if method == "z_gettreestate" {
		return nil, errors.New("method not found")
	}
	if method != "z_gettreestatelegacy" {
		testT.Fatal("unexpected method in z_gettreestatelegacyStub:", method)
	}

	// Mock response that matches TreasureChest legacy format (no 'active' field, no Orchard)
	mockResponse := `{
		"hash": "0000000001234567890123456789012345678901234567890123456789abcdef",
		"height": 100200,
		"time": 1609459200,
		"sprout": {
			"commitments": {
				"finalRoot": "1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef",
				"finalState": "sprout1234567890abcdef1234567890abcdef1234567890abcdef1234567890ab"
			}
		},
		"sapling": {
			"commitments": {
				"finalRoot": "abcd1234567890abcdef1234567890abcdef1234567890abcdef1234567890ab",
				"finalState": "abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef"
			}
		}
	}`

	return json.RawMessage(mockResponse), nil
}

func TestGetTreeState(t *testing.T) {
	testT = t
	common.RawRequest = z_gettreestatelegacyStub
	lwd := newTreeStateTestStreamer(t)

	// Test with height
	blockID := &walletrpc.BlockID{Height: 100200}
	treeState, err := lwd.GetTreeState(context.Background(), blockID)
	if err != nil {
		t.Fatal("GetTreeState failed with height:", err)
	}

	// Verify the response
	if treeState.Network != "main" {
		t.Fatal("Unexpected network:", treeState.Network)
	}
	if treeState.Height != 100200 {
		t.Fatal("Unexpected height:", treeState.Height)
	}
	if treeState.Hash != "0000000001234567890123456789012345678901234567890123456789abcdef" {
		t.Fatal("Unexpected hash:", treeState.Hash)
	}
	if treeState.Time != 1609459200 {
		t.Fatal("Unexpected time:", treeState.Time)
	}
	// Check that we're using finalState when available
	if treeState.SaplingTree != "abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef" {
		t.Fatal("Unexpected sapling tree (should use finalState):", treeState.SaplingTree)
	}
	// Legacy format does not support Ironwood.
	if treeState.IronwoodTree != "" {
		t.Fatal("IronwoodTree should be empty for legacy format:", treeState.IronwoodTree)
	}

	// Test with hash
	hashBytes := []byte{0x00, 0x00, 0x00, 0x00, 0x01, 0x23, 0x45, 0x67, 0x89, 0x01, 0x23, 0x45, 0x67, 0x89, 0x01, 0x23, 0x45, 0x67, 0x89, 0x01, 0x23, 0x45, 0x67, 0x89, 0x01, 0x23, 0x45, 0x67, 0x89, 0xab, 0xcd, 0xef}
	blockID = &walletrpc.BlockID{Hash: hashBytes}
	treeState, err = lwd.GetTreeState(context.Background(), blockID)
	if err != nil {
		t.Fatal("GetTreeState failed with hash:", err)
	}

	// Should get the same result
	if treeState.Height != 100200 {
		t.Fatal("Unexpected height with hash lookup:", treeState.Height)
	}
}

func z_gettreestateStubFallbackToRoot(method string, params []json.RawMessage) (json.RawMessage, error) {
	if method == "z_gettreestate" {
		return nil, errors.New("method not found")
	}
	if method != "z_gettreestatelegacy" {
		testT.Fatal("unexpected method in z_gettreestateStubFallbackToRoot:", method)
	}

	// Mock response where finalState is empty, should fallback to finalRoot
	mockResponse := `{
		"hash": "0000000001234567890123456789012345678901234567890123456789abcdef",
		"height": 100200,
		"time": 1609459200,
		"sprout": {
			"commitments": {
				"finalRoot": "1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef"
			}
		},
		"sapling": {
			"commitments": {
				"finalRoot": "abcd1234567890abcdef1234567890abcdef1234567890abcdef1234567890ab"
			}
		}
	}`

	return json.RawMessage(mockResponse), nil
}

func TestGetTreeStateFallbackToRoot(t *testing.T) {
	testT = t
	common.RawRequest = z_gettreestateStubFallbackToRoot
	lwd := newTreeStateTestStreamer(t)

	blockID := &walletrpc.BlockID{Height: 100200}
	treeState, err := lwd.GetTreeState(context.Background(), blockID)
	if err != nil {
		t.Fatal("GetTreeState failed:", err)
	}

	// Check that we fallback to finalRoot when finalState is not available
	if treeState.SaplingTree != "abcd1234567890abcdef1234567890abcdef1234567890abcdef1234567890ab" {
		t.Fatal("Unexpected sapling tree (should use finalRoot):", treeState.SaplingTree)
	}
	// Legacy format does not support Ironwood.
	if treeState.IronwoodTree != "" {
		t.Fatal("IronwoodTree should be empty for legacy format:", treeState.IronwoodTree)
	}
}

func TestGetTreeStateErrors(t *testing.T) {
	testT = t
	lwd := newTreeStateTestStreamer(t)

	// Test with no identifier
	blockID := &walletrpc.BlockID{}
	_, err := lwd.GetTreeState(context.Background(), blockID)
	if err == nil {
		t.Fatal("GetTreeState should have failed with no identifier")
	}
	if err.Error() != "request for unspecified identifier" {
		t.Fatal("Unexpected error message:", err.Error())
	}
}

// Tests for GetBridgeTreeState (new bridge trees format)

func z_gettreestateBridgeStub(method string, params []json.RawMessage) (json.RawMessage, error) {
	if method != "z_gettreestate" {
		testT.Fatal("unexpected method in z_gettreestateBridgeStub:", method)
	}

	// Mock response that matches TreasureChest new format with bridge trees
	mockResponse := `{
		"hash": "0000000001234567890123456789012345678901234567890123456789abcdef",
		"height": 100200,
		"time": 1609459200,
		"sprout": {
			"active": true,
			"commitments": {
				"finalRoot": "1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef",
				"finalState": "sprout1234567890abcdef1234567890abcdef1234567890abcdef1234567890ab"
			}
		},
		"sapling": {
			"active": true,
			"commitments": {
				"finalRoot": "abcd1234567890abcdef1234567890abcdef1234567890abcdef1234567890ab",
				"finalState": "abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef"
			}
		},
		"orchard": {
			"active": true,
			"commitments": {
				"finalRoot": "ef123456789abcdef123456789abcdef123456789abcdef123456789abcdef12",
				"finalState": "1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234"
			}
		}
	}`

	return json.RawMessage(mockResponse), nil
}

func TestGetBridgeTreeState(t *testing.T) {
	testT = t
	common.RawRequest = z_gettreestateBridgeStub
	lwd := newTreeStateTestStreamer(t)

	// Test with height
	blockID := &walletrpc.BlockID{Height: 100200}
	treeState, err := lwd.GetBridgeTreeState(context.Background(), blockID)
	if err != nil {
		t.Fatal("GetBridgeTreeState failed with height:", err)
	}

	// Verify the response
	if treeState.Network != "main" {
		t.Fatal("Unexpected network:", treeState.Network)
	}
	if treeState.Height != 100200 {
		t.Fatal("Unexpected height:", treeState.Height)
	}
	if treeState.Hash != "0000000001234567890123456789012345678901234567890123456789abcdef" {
		t.Fatal("Unexpected hash:", treeState.Hash)
	}
	if treeState.Time != 1609459200 {
		t.Fatal("Unexpected time:", treeState.Time)
	}
	// Check that we're using finalState when available
	if treeState.SaplingTree != "abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef" {
		t.Fatal("Unexpected sapling tree (should use finalState):", treeState.SaplingTree)
	}
	if treeState.IronwoodTree != "1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234" {
		t.Fatal("Unexpected Ironwood tree (should use finalState):", treeState.IronwoodTree)
	}

	// Test with hash
	hashBytes := []byte{0x00, 0x00, 0x00, 0x00, 0x01, 0x23, 0x45, 0x67, 0x89, 0x01, 0x23, 0x45, 0x67, 0x89, 0x01, 0x23, 0x45, 0x67, 0x89, 0x01, 0x23, 0x45, 0x67, 0x89, 0x01, 0x23, 0x45, 0x67, 0x89, 0xab, 0xcd, 0xef}
	blockID = &walletrpc.BlockID{Hash: hashBytes}
	treeState, err = lwd.GetBridgeTreeState(context.Background(), blockID)
	if err != nil {
		t.Fatal("GetBridgeTreeState failed with hash:", err)
	}

	// Should get the same result
	if treeState.Height != 100200 {
		t.Fatal("Unexpected height with hash lookup:", treeState.Height)
	}
}

func z_gettreestateBridgeStubFallbackToRoot(method string, params []json.RawMessage) (json.RawMessage, error) {
	if method != "z_gettreestate" {
		testT.Fatal("unexpected method in z_gettreestateBridgeStubFallbackToRoot:", method)
	}

	// Mock response where finalState is empty, should fallback to finalRoot
	mockResponse := `{
		"hash": "0000000001234567890123456789012345678901234567890123456789abcdef",
		"height": 100200,
		"time": 1609459200,
		"sprout": {
			"active": true,
			"commitments": {
				"finalRoot": "1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef"
			}
		},
		"sapling": {
			"active": true,
			"commitments": {
				"finalRoot": "abcd1234567890abcdef1234567890abcdef1234567890abcdef1234567890ab"
			}
		},
		"orchard": {
			"active": true,
			"commitments": {
				"finalRoot": "ef123456789abcdef123456789abcdef123456789abcdef123456789abcdef12"
			}
		}
	}`

	return json.RawMessage(mockResponse), nil
}

func TestGetBridgeTreeStateFallbackToRoot(t *testing.T) {
	testT = t
	common.RawRequest = z_gettreestateBridgeStubFallbackToRoot
	lwd := newTreeStateTestStreamer(t)

	blockID := &walletrpc.BlockID{Height: 100200}
	treeState, err := lwd.GetBridgeTreeState(context.Background(), blockID)
	if err != nil {
		t.Fatal("GetBridgeTreeState failed:", err)
	}

	// Check that we fallback to finalRoot when finalState is not available
	if treeState.SaplingTree != "abcd1234567890abcdef1234567890abcdef1234567890abcdef1234567890ab" {
		t.Fatal("Unexpected sapling tree (should use finalRoot):", treeState.SaplingTree)
	}
	if treeState.IronwoodTree != "ef123456789abcdef123456789abcdef123456789abcdef123456789abcdef12" {
		t.Fatal("Unexpected Ironwood tree (should use finalRoot):", treeState.IronwoodTree)
	}
}

func TestGetBridgeTreeStateErrors(t *testing.T) {
	testT = t
	lwd := newTreeStateTestStreamer(t)

	// Test with no identifier
	blockID := &walletrpc.BlockID{}
	_, err := lwd.GetBridgeTreeState(context.Background(), blockID)
	if err == nil {
		t.Fatal("GetBridgeTreeState should have failed with no identifier")
	}
	if err.Error() != "request for unspecified identifier" {
		t.Fatal("Unexpected error message:", err.Error())
	}
}
