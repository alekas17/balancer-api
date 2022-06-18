package models

import (
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/latoken/bridge-balancer-service/src/service/storage"
)

// ServiceConfig contains configurations for rest-api service.
type ServiceConfig struct {
	Host string // Service Host
	Port string // Service port
}

// RelayerStatus ...
type RelayerStatus struct {
	Mode    string                  `json:"mode"`
	Workers map[string]WorkerStatus `json:"workers"`
}

// WorkerStatus ...
type WorkerStatus struct {
	Height             int64         `json:"height"`
	SyncHeight         int64         `json:"sync_height"`
	LastBlockFetchedAt time.Time     `json:"last_block_fetched_at"`
	Status             interface{}   `json:"status"`
	Account            WorkerAccount `json:"account"`
}

// WorkerAccount ...
type WorkerAccount struct {
	Address string `json:"address"`
}

// BlockAndTxLogs ...
type BlockAndTxLogs struct {
	Height          int64
	BlockHash       string
	ParentBlockHash string
	BlockTime       int64
	TxLogs          []*storage.TxLog
}

// SwapRequest ...
type SwapRequest struct {
	ID                   common.Hash
	RandomNumberHash     common.Hash
	ExpireHeight         int64
	SenderAddress        string
	RecipientAddress     string
	RecipientWorkerChain string
	OutAmount            *big.Int
}

// StorageConfig contains configurations for storage, postgreSQL
type StorageConfig struct {
	URL        string // DataBase URL for connection
	DBDriver   string // DataBase driver
	DBHOST     string // DataBase host
	DBPORT     int64
	DBSSL      string // DataBase sslmode
	DBName     string // DataBase name
	DBUser     string // DataBase's user
	DBPassword string // User's password
}

// WorkerConfig ...
type WorkerConfig struct {
	NetworkType                    string         `json:"type"`
	ChainName                      string         `json:"chain_id"`
	User                           string         `json:"user"`
	Password                       string         `json:"password"`
	SwapType                       string         `json:"swap_type"`
	KeyType                        string         `json:"key_type"`
	AWSRegion                      string         `json:"aws_region"`
	AWSSecretName                  string         `json:"aws_secret_name"`
	PrivateKey                     string         `json:"private_key"`
	Provider                       string         `json:"provider"`
	ContractAddr                   common.Address `json:"contract_addr"`
	TokenContractAddr              common.Address `json:"token_contract_addr"`
	WorkerAddr                     common.Address `json:"worker_addr"`
	ColdWalletAddr                 common.Address `json:"cold_wallet_addr"`
	TokenBalanceAlertThreshold     *big.Int       `json:"token_balance_alert_threshold"`
	EthBalanceAlertThreshold       *big.Int       `json:"eth_balance_alert_threshold"`
	AllowanceBalanceAlertThreshold *big.Int       `json:"allowance_balance_alert_threshold"`
	FetchInterval                  int64          `json:"fetch_interval"`
	GasLimit                       int64          `json:"gas_limit"`
	GasPrice                       *big.Int       `json:"gas_price"`
	ChainDecimal                   int            `json:"chain_decimal"`
	ConfirmNum                     int64          `json:"confirm_num"`
	StartBlockHeight               int64          `json:"start_block_height"`
	DestinationChainID             string         `json:"dest_id"`
}

type FetcherConfig struct {
	AllTokens []string
}

type TssConfig struct {
	Address    string
	BaseFolder string
}

// RelayerConfig ...
type RelayerConfig struct {
	Address common.Address
}

// // ResourceID
// type ResourceId struct {
// 	Name string
// 	ID   string
// }

// type PriceConfig struct {
// 	Name string
// }
