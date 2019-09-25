package common

import (
	"encoding/hex"
	"encoding/json"
	"strconv"
	"strings"

	"github.com/adityapk00/lightwalletd/parser"
	"github.com/adityapk00/lightwalletd/walletrpc"
	"github.com/btcsuite/btcd/rpcclient"
	"github.com/pkg/errors"
)

func GetSaplingInfo(rpcClient *rpcclient.Client) (int, string, error) {
	result, rpcErr := rpcClient.RawRequest("getblockchaininfo", make([]json.RawMessage, 0))

	var err error
	var errCode int64

	// For some reason, the error responses are not JSON
	if rpcErr != nil {
		errParts := strings.SplitN(rpcErr.Error(), ":", 2)
		errCode, err = strconv.ParseInt(errParts[0], 10, 32)
		//Check to see if we are requesting a height the zcashd doesn't have yet
		if err == nil && errCode == -8 {
			return -1, "", nil
		}
		return -1, "", errors.Wrap(rpcErr, "error requesting block")
	}

	var f interface{}
	err = json.Unmarshal(result, &f)
	if err != nil {
		return -1, "", errors.Wrap(err, "error reading JSON response")
	}

	chainName := f.(map[string]interface{})["chain"].(string)

	upgradeJSON := f.(map[string]interface{})["upgrades"]
	saplingJSON := upgradeJSON.(map[string]interface{})["76b809bb"] // Sapling ID
	saplingHeight := saplingJSON.(map[string]interface{})["activationheight"].(float64)

	return int(saplingHeight), chainName, nil
}

func GetBlock(rpcClient *rpcclient.Client, height int) (*parser.Block, error) {
	params := make([]json.RawMessage, 2)
	params[0] = json.RawMessage("\"" + strconv.Itoa(height) + "\"")
	params[1] = json.RawMessage("0")
	result, rpcErr := rpcClient.RawRequest("getblock", params)

	var err error
	var errCode int64

	// For some reason, the error responses are not JSON
	if rpcErr != nil {
		errParts := strings.SplitN(rpcErr.Error(), ":", 2)
		errCode, err = strconv.ParseInt(errParts[0], 10, 32)
		//Check to see if we are requesting a height the zcashd doesn't have yet
		if err == nil && errCode == -8 {
			return nil, nil
		}
		return nil, errors.Wrap(rpcErr, "error requesting block")
	}

	var blockDataHex string
	err = json.Unmarshal(result, &blockDataHex)
	if err != nil {
		return nil, errors.Wrap(err, "error reading JSON response")
	}

	blockData, err := hex.DecodeString(blockDataHex)
	if err != nil {
		return nil, errors.Wrap(err, "error decoding getblock output")
	}

	block := parser.NewBlock()
	rest, err := block.ParseFromSlice(blockData)
	if err != nil {
		return nil, errors.Wrap(err, "error parsing block")
	}
	if len(rest) != 0 {
		return nil, errors.New("received overlong message")
	}
	return block, nil
}

func GetBlockRange(rpcClient *rpcclient.Client, blockOut chan<- walletrpc.CompactBlock,
	errOut chan<- error, start, end int) {

	// Go over [start, end] inclusive
	for i := start; i <= end; i++ {
		block, err := GetBlock(rpcClient, i)
		if err != nil {
			errOut <- err
			return
		}

		blockOut <- *block.ToCompact()
	}

	errOut <- nil
}
