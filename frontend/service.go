package frontend

import (
	"context"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/adityapk00/btcd/rpcclient"
	"github.com/golang/protobuf/proto"
	"github.com/sirupsen/logrus"

	// blank import for sqlite driver support
	_ "github.com/mattn/go-sqlite3"

	"github.com/adityapk00/lightwalletd/common"
	"github.com/adityapk00/lightwalletd/storage"
	"github.com/adityapk00/lightwalletd/walletrpc"
)

var (
	ErrUnspecified = errors.New("request for unspecified identifier")
)

// the service type
type SqlStreamer struct {
	db     *sql.DB
	client *rpcclient.Client
	log    *logrus.Entry
}

func NewSQLiteStreamer(dbPath string, client *rpcclient.Client, log *logrus.Entry) (walletrpc.CompactTxStreamerServer, error) {
	db, err := sql.Open("sqlite3", fmt.Sprintf("file:%s?_busy_timeout=10000&cache=shared", dbPath))
	db.SetMaxOpenConns(1)
	if err != nil {
		return nil, err
	}

	// Creates our tables if they don't already exist.
	err = storage.CreateTables(db)
	if err != nil {
		return nil, err
	}

	return &SqlStreamer{db, client, log}, nil
}

func (s *SqlStreamer) GracefulStop() error {
	return s.db.Close()
}

func (s *SqlStreamer) GetLatestBlock(ctx context.Context, placeholder *walletrpc.ChainSpec) (*walletrpc.BlockID, error) {
	// the ChainSpec type is an empty placeholder
	height, err := storage.GetCurrentHeight(ctx, s.db)
	if err != nil {
		return nil, err
	}
	// TODO: also return block hashes here
	return &walletrpc.BlockID{Height: uint64(height)}, nil
}

func (s *SqlStreamer) GetBlock(ctx context.Context, id *walletrpc.BlockID) (*walletrpc.CompactBlock, error) {
	if id.Height == 0 && id.Hash == nil {
		return nil, ErrUnspecified
	}

	var blockBytes []byte
	var err error

	// Precedence: a hash is more specific than a height. If we have it, use it first.
	if id.Hash != nil {
		leHashString := hex.EncodeToString(id.Hash)
		blockBytes, err = storage.GetBlockByHash(ctx, s.db, leHashString)
	} else {
		blockBytes, err = storage.GetBlock(ctx, s.db, int(id.Height))
	}

	if err != nil {
		return nil, err
	}

	cBlock := &walletrpc.CompactBlock{}
	err = proto.Unmarshal(blockBytes, cBlock)
	return cBlock, err
}

func (s *SqlStreamer) GetAddressTxids(addressBlockFilter *walletrpc.TransparentAddressBlockFilter, resp walletrpc.CompactTxStreamer_GetAddressTxidsServer) error {
	params := make([]json.RawMessage, 1)
	st := "{\"addresses\": [\"" + addressBlockFilter.Address + "\"]," +
		"\"start\": " + strconv.FormatUint(addressBlockFilter.Range.Start.Height, 10) +
		", \"end\": " + strconv.FormatUint(addressBlockFilter.Range.End.Height, 10) + "}"

	params[0] = json.RawMessage(st)

	result, rpcErr := s.client.RawRequest("getaddresstxids", params)

	var err error
	var errCode int64

	// For some reason, the error responses are not JSON
	if rpcErr != nil {
		s.log.Errorf("Got error: %s", rpcErr.Error())
		errParts := strings.SplitN(rpcErr.Error(), ":", 2)
		errCode, err = strconv.ParseInt(errParts[0], 10, 32)
		//Check to see if we are requesting a height the zcashd doesn't have yet
		if err == nil && errCode == -8 {
			return nil
		}
		return nil
	}

	var txids []string
	err = json.Unmarshal(result, &txids)
	if err != nil {
		s.log.Errorf("Got error: %s", err.Error())
		return nil
	}

	timeout, cancel := context.WithTimeout(resp.Context(), 30*time.Second)
	defer cancel()

	for _, txidstr := range txids {
		txid, _ := hex.DecodeString(txidstr)
		// Txid is read as a string, which is in big-endian order. But when converting
		// to bytes, it should be little-endian
		for left, right := 0, len(txid)-1; left < right; left, right = left+1, right-1 {
			txid[left], txid[right] = txid[right], txid[left]
		}

		tx, err := s.GetTransaction(timeout, &walletrpc.TxFilter{Hash: txid})
		if err != nil {
			s.log.Errorf("Got error: %s", err.Error())
			return nil
		}

		resp.Send(tx)
	}

	return nil
}

func (s *SqlStreamer) GetUtxos(address *walletrpc.TransparentAddress, resp walletrpc.CompactTxStreamer_GetUtxosServer) error {
	utxos, _ := s.client.GetAddressUtxos(address.Address)

	for _, utxo := range utxos {
		txid, _ := hex.DecodeString(utxo.TxID)
		// Txid is read as a string, which is in big-endian order. But when converting
		// to bytes, it should be little-endian
		for left, right := 0, len(txid)-1; left < right; left, right = left+1, right-1 {
			txid[left], txid[right] = txid[right], txid[left]
		}

		script, _ := hex.DecodeString(utxo.ScriptPubKey)

		respUtxo := &walletrpc.Utxo{
			Address:     address,
			Txid:        txid,
			OutputIndex: uint64(utxo.OutputIndex),
			Script:      script,
			Value:       utxo.Amount,
			Height:      uint64(utxo.Height),
		}

		err := resp.Send(respUtxo)
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *SqlStreamer) GetBlockRange(span *walletrpc.BlockRange, resp walletrpc.CompactTxStreamer_GetBlockRangeServer) error {
	blockChan := make(chan []byte)
	errChan := make(chan error)

	// TODO configure or stress-test this timeout
	timeout, cancel := context.WithTimeout(resp.Context(), 30*time.Second)
	defer cancel()
	go storage.GetBlockRange(timeout,
		s.db,
		blockChan,
		errChan,
		int(span.Start.Height),
		int(span.End.Height),
	)

	for {
		select {
		case err := <-errChan:
			// this will also catch context.DeadlineExceeded from the timeout
			return err
		case blockBytes := <-blockChan:
			cBlock := &walletrpc.CompactBlock{}
			err := proto.Unmarshal(blockBytes, cBlock)
			if err != nil {
				return err // TODO really need better logging in this whole service
			}
			err = resp.Send(cBlock)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (s *SqlStreamer) GetTransaction(ctx context.Context, txf *walletrpc.TxFilter) (*walletrpc.RawTransaction, error) {
	var txBytes []byte
	var txHeight int
	var err error

	if txf.Hash != nil {
		leHashString := hex.EncodeToString(txf.Hash)
		txBytes, txHeight, err = storage.GetTxByHash(ctx, s.db, leHashString)
		if err != nil {
			return nil, err
		}
		return &walletrpc.RawTransaction{Data: txBytes, Height: uint64(txHeight)}, nil

	}

	if txf.Block.Hash != nil {
		leHashString := hex.EncodeToString(txf.Hash)
		txBytes, txHeight, err = storage.GetTxByHashAndIndex(ctx, s.db, leHashString, int(txf.Index))
		if err != nil {
			return nil, err
		}
		return &walletrpc.RawTransaction{Data: txBytes, Height: uint64(txHeight)}, nil
	}

	// A totally unset protobuf will attempt to fetch the genesis coinbase tx.
	txBytes, txHeight, err = storage.GetTxByHeightAndIndex(ctx, s.db, int(txf.Block.Height), int(txf.Index))
	if err != nil {
		return nil, err
	}
	return &walletrpc.RawTransaction{Data: txBytes, Height: uint64(txHeight)}, nil
}

// GetLightdInfo gets the LightWalletD (this server) info
func (s *SqlStreamer) GetLightdInfo(ctx context.Context, in *walletrpc.Empty) (*walletrpc.LightdInfo, error) {
	saplingHeight, chainName, err := common.GetSaplingInfo(s.client)

	if err != nil {
		s.log.WithFields(logrus.Fields{
			"error": err,
		}).Warn("Unable to get sapling activation height")
	}

	// TODO these are called Error but they aren't at the moment.
	// A success will return code 0 and message txhash.
	return &walletrpc.LightdInfo{
		Version:                 "0.1-zeclightd",
		Vendor:                  "ZecWallet LightWalletD",
		TaddrSupport:            true,
		ChainName:               chainName,
		SaplingActivationHeight: uint64(saplingHeight),
	}, nil
}

// SendTransaction forwards raw transaction bytes to a zcashd instance over JSON-RPC
func (s *SqlStreamer) SendTransaction(ctx context.Context, rawtx *walletrpc.RawTransaction) (*walletrpc.SendResponse, error) {
	// sendrawtransaction "hexstring" ( allowhighfees )
	//
	// Submits raw transaction (serialized, hex-encoded) to local node and network.
	//
	// Also see createrawtransaction and signrawtransaction calls.
	//
	// Arguments:
	// 1. "hexstring"    (string, required) The hex string of the raw transaction)
	// 2. allowhighfees    (boolean, optional, default=false) Allow high fees
	//
	// Result:
	// "hex"             (string) The transaction hash in hex

	// Construct raw JSON-RPC params
	params := make([]json.RawMessage, 1)
	txHexString := hex.EncodeToString(rawtx.Data)
	params[0] = json.RawMessage("\"" + txHexString + "\"")
	result, rpcErr := s.client.RawRequest("sendrawtransaction", params)

	var err error
	var errCode int64
	var errMsg string

	// For some reason, the error responses are not JSON
	if rpcErr != nil {
		errParts := strings.SplitN(rpcErr.Error(), ":", 2)
		errMsg = strings.TrimSpace(errParts[1])
		errCode, err = strconv.ParseInt(errParts[0], 10, 32)
		if err != nil {
			// This should never happen. We can't panic here, but it's that class of error.
			// This is why we need integration testing to work better than regtest currently does. TODO.
			return nil, errors.New("SendTransaction couldn't parse error code")
		}
	} else {
		errMsg = string(result)
	}

	// TODO these are called Error but they aren't at the moment.
	// A success will return code 0 and message txhash.
	return &walletrpc.SendResponse{
		ErrorCode:    int32(errCode),
		ErrorMessage: errMsg,
	}, nil
}
