package main

import (
	"context"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/adityapk00/btcd/rpcclient"
	"github.com/golang/protobuf/proto"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"github.com/adityapk00/lightwalletd/common"
	"github.com/adityapk00/lightwalletd/frontend"
	"github.com/adityapk00/lightwalletd/parser"
	"github.com/adityapk00/lightwalletd/storage"
)

var log *logrus.Entry
var logger = logrus.New()
var db *sql.DB

// Options is a struct holding command line options
type Options struct {
	dbPath        string
	logLevel      uint64
	logPath       string
	zcashConfPath string
}

func main() {
	opts := &Options{}
	flag.StringVar(&opts.dbPath, "db-path", "", "the path to a sqlite database file")
	flag.Uint64Var(&opts.logLevel, "log-level", uint64(logrus.InfoLevel), "log level (logrus 1-7)")
	flag.StringVar(&opts.logPath, "log-file", "", "log file to write to")
	flag.StringVar(&opts.zcashConfPath, "conf-file", "", "conf file to pull RPC creds from")
	// TODO prod metrics
	// TODO support config from file and env vars
	flag.Parse()

	if opts.dbPath == "" {
		flag.Usage()
		os.Exit(1)
	}

	// Initialize logging
	logger.SetFormatter(&logrus.TextFormatter{
		//DisableColors: true,
		FullTimestamp:          true,
		DisableLevelTruncation: true,
	})

	if opts.logPath != "" {
		// instead write parsable logs for logstash/splunk/etc
		output, err := os.OpenFile(opts.logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			log.WithFields(logrus.Fields{
				"error": err,
				"path":  opts.logPath,
			}).Fatal("couldn't open log file")
		}
		defer output.Close()
		logger.SetOutput(output)
		logger.SetFormatter(&logrus.JSONFormatter{})
	}

	logger.SetLevel(logrus.Level(opts.logLevel))

	log = logger.WithFields(logrus.Fields{
		"app": "lightwd",
	})

	// Initialize database
	db, err := sql.Open("sqlite3", fmt.Sprintf("file:%s?_busy_timeout=10000&cache=shared", opts.dbPath))
	db.SetMaxOpenConns(1)
	if err != nil {
		log.WithFields(logrus.Fields{
			"db_path": opts.dbPath,
			"error":   err,
		}).Fatal("couldn't open SQL db")
	}

	// Creates our tables if they don't already exist.
	err = storage.CreateTables(db)
	if err != nil {
		log.WithFields(logrus.Fields{
			"error": err,
		}).Fatal("couldn't create SQL tables")
	}

	//Initialize RPC connection with full node zcashd
	rpcClient, err := frontend.NewZRPCFromConf(opts.zcashConfPath)
	if err != nil {
		log.WithFields(logrus.Fields{
			"error": err,
		}).Warn("zcash.conf failed, will try empty credentials for rpc")

		//Default to testnet, but user MUST specify rpcuser and rpcpassword in zcash.conf; no default
		rpcClient, err = frontend.NewZRPCFromCreds("127.0.0.1:18232", "", "")

		if err != nil {
			log.WithFields(logrus.Fields{
				"error": err,
			}).Fatal("couldn't start rpc connection")
		}
	}

	ctx := context.Background()
	height, err := storage.GetCurrentHeight(ctx, db)
	if err != nil {
		log.WithFields(logrus.Fields{
			"error": err,
		}).Warn("Unable to get current height from local db storage. This is OK if you're starting this for the first time.")
	}

	// Get the sapling activation height from the RPC
	saplingHeight, chainName, err := common.GetSaplingInfo(rpcClient)
	if err != nil {
		log.WithFields(logrus.Fields{
			"error": err,
		}).Warn("Unable to get sapling activation height")
	}

	log.WithField("saplingHeight", saplingHeight).Info("Got sapling height ", saplingHeight, " chain ", chainName)

	//ingest from Sapling testnet height
	if height < saplingHeight {
		height = saplingHeight
		log.WithFields(logrus.Fields{
			"error": err,
		}).Warn("invalid current height read from local db storage")
	}

	timeoutCount := 0
	reorgCount := -1
	hash := ""
	phash := ""
	// Start listening for new blocks
	for {
		if reorgCount > 0 {
			reorgCount = -1
			height -= 10
		}
		block, err := getBlock(rpcClient, height)

		if err != nil {
			log.WithFields(logrus.Fields{
				"height": height,
				"error":  err,
			}).Warn("error with getblock")
			timeoutCount++
			if timeoutCount == 3 {
				log.WithFields(logrus.Fields{
					"timeouts": timeoutCount,
				}).Warn("unable to issue RPC call to zcashd node 3 times")
				break
			}
		}
		if block != nil {
			handleBlock(db, block)
			if timeoutCount > 0 {
				timeoutCount--
			}
			phash = hex.EncodeToString(block.GetPrevHash())
			//check for reorgs once we have inital block hash from startup
			if hash != phash && reorgCount != -1 {
				reorgCount++
				log.WithFields(logrus.Fields{
					"height": height,
					"hash":   hash,
					"phash":  phash,
					"reorg":  reorgCount,
				}).Warn("REORG")
			} else {
				hash = hex.EncodeToString(block.GetDisplayHash())
			}
			if reorgCount == -1 {
				hash = hex.EncodeToString(block.GetDisplayHash())
				reorgCount = 0
			}
			height++
		} else {
			//TODO implement blocknotify to minimize polling on corner cases
			time.Sleep(60 * time.Second)
		}
	}
}

func getBlock(rpcClient *rpcclient.Client, height int) (*parser.Block, error) {
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

func handleBlock(db *sql.DB, block *parser.Block) {
	prevBlockHash := hex.EncodeToString(block.GetPrevHash())
	blockHash := hex.EncodeToString(block.GetEncodableHash())
	marshaledBlock, _ := proto.Marshal(block.ToCompact())

	err := storage.StoreBlock(
		db,
		block.GetHeight(),
		prevBlockHash,
		blockHash,
		block.HasSaplingTransactions(),
		marshaledBlock,
	)

	entry := log.WithFields(logrus.Fields{
		"block_height":  block.GetHeight(),
		"block_hash":    hex.EncodeToString(block.GetDisplayHash()),
		"prev_hash":     hex.EncodeToString(block.GetDisplayPrevHash()),
		"block_version": block.GetVersion(),
		"tx_count":      block.GetTxCount(),
		"sapling":       block.HasSaplingTransactions(),
		"error":         err,
	})

	if err != nil {
		entry.Error("new block")
	} else {
		entry.Info("new block")
	}

	for index, tx := range block.Transactions() {
		txHash := hex.EncodeToString(tx.GetEncodableHash())
		err = storage.StoreTransaction(
			db,
			block.GetHeight(),
			blockHash,
			index,
			txHash,
			tx.Bytes(),
		)
		entry = log.WithFields(logrus.Fields{
			"block_height": block.GetHeight(),
			"block_hash":   hex.EncodeToString(block.GetDisplayHash()),
			"tx_index":     index,
			"tx_size":      len(tx.Bytes()),
			"sapling":      tx.HasSaplingTransactions(),
			"error":        err,
		})
		if err != nil {
			entry.Error("storing tx")
		} else {
			entry.Debug("storing tx")
		}
	}
}
