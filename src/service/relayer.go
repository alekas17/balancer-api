package rlr

import (
	"sync"
	"time"

	watcher "github.com/latoken/bridge-balancer-service/src/service/blockchains-watcher"
	fetcher "github.com/latoken/bridge-balancer-service/src/service/price-fetcher"
	"github.com/latoken/bridge-balancer-service/src/service/storage"
	workers "github.com/latoken/bridge-balancer-service/src/service/workers"
	"github.com/latoken/bridge-balancer-service/src/service/workers/eth-compatible"

	"github.com/jinzhu/gorm"
	"github.com/sirupsen/logrus"

	"github.com/latoken/bridge-balancer-service/src/models"
)

// BridgeSRV ...
type BridgeSRV struct {
	sync.RWMutex
	logger   *logrus.Logger
	Watcher  *watcher.WatcherSRV
	laWorker workers.IWorker
	Workers  map[string]workers.IWorker
	storage  *storage.DataBase
	Fetcher  *fetcher.FetcherSrv
}

// CreateNewBridgeSRV ...
func CreateNewBridgeSRV(logger *logrus.Logger, gormDB *gorm.DB, laConfig *models.WorkerConfig,
	chainCfgs []*models.WorkerConfig, fetCfg *models.FetcherConfig, resourceIDs []*storage.ResourceId) *BridgeSRV {
	// init database
	db, err := storage.InitStorage(gormDB)
	if err != nil {
		logger.Fatalf("Connect to DataBase: ", err)
	}

	// create Relayer instance
	inst := BridgeSRV{
		logger:   logger,
		storage:  db,
		laWorker: eth.NewErc20Worker(logger, laConfig),
		Workers:  make(map[string]workers.IWorker),
	}
	// create erc20 worker
	for _, cfg := range chainCfgs {
		inst.Workers[cfg.ChainName] = eth.NewErc20Worker(logger, cfg)
	}

	// check rules for workers(>=2, different chainIDs...)
	if len(inst.Workers) < 1 {
		logger.Fatalf("Num of workers must be > 1, but = %d", len(inst.Workers))
		return nil
	}
	inst.Watcher = watcher.CreateNewWatcherSRV(logger, db, inst.Workers)
	inst.Fetcher = fetcher.CreateNewFetcherSrv(logger, db, fetCfg)

	// create la worker
	inst.Workers["LA"] = inst.laWorker

	db.SaveResourceIDs(resourceIDs)
	return &inst
}

// !!! TODO !!!

// Run ...
func (r *BridgeSRV) Run() {
	// start watcher
	r.Watcher.Run()
	r.Fetcher.Run()
	// run Worker workers
	for _, worker := range r.Workers {
		go r.ConfirmWorkerTx(worker)
		go r.CheckTxSentRoutine(worker)
		if worker.GetChainName() == "LA" {
			go r.emitFeeTransfer(worker)
		}
	}
}

// ConfirmWorkerTx ...
func (r *BridgeSRV) ConfirmWorkerTx(worker workers.IWorker) {
	for {
		txLogs, err := r.storage.FindTxLogs(worker.GetChainName(), worker.GetConfirmNum())
		if err != nil {
			r.logger.Errorf("ConfirmWorkerTx(), err = %s", err)
			time.Sleep(10 * time.Second)
			continue
		}

		txHashes := make([]string, 0, len(txLogs))
		newEvents := make([]*storage.Event, 0)

		for _, txLog := range txLogs {
			if txLog.TxType == storage.TxTypeFeeTransfer {
				r.logger.Infoln("New Event")
				newEvent := &storage.Event{
					ReceiverAddr:       txLog.ReceiverAddr,
					ChainID:            txLog.Chain,
					DestinationChainID: txLog.DestinationChainID,
					OriginChainID:      txLog.OriginСhainID,
					InAmount:           txLog.InAmount,
					ResourceID:         txLog.ResourceID,
					DepositNonce:       txLog.DepositNonce,
					Height:             txLog.Height,
					SwapID:             txLog.SwapID,
					Status:             storage.EventStatusFeeTransferInitConfrimed,
					CreateTime:         time.Now().Unix(),
				}
				newEvents = append(newEvents, newEvent)
			}
			txHashes = append(txHashes, txLog.TxHash)
		}

		//
		if err := r.storage.ConfirmWorkerTx(worker.GetChainName(), txLogs, txHashes, newEvents); err != nil {
			r.logger.Errorf("compensate new swap tx error, err=%s", err)
		}

		time.Sleep(2 * time.Second)
	}
}

// CheckTxSentRoutine ...
func (r *BridgeSRV) CheckTxSentRoutine(worker workers.IWorker) {
	for {
		r.CheckTxSent(worker)
		time.Sleep(time.Second)
	}
}

// CheckTxSent ...
func (r *BridgeSRV) CheckTxSent(worker workers.IWorker) {
	txsSent, err := r.storage.GetTxsSentByStatus(worker.GetChainName())
	if err != nil {
		r.logger.WithFields(logrus.Fields{"function": "CheckTxSent() | GetTxsSentByStatus()"}).Errorln(err)
		return
	}

	for _, txSent := range txsSent {
		// Get status of tx from chain
		status := worker.GetSentTxStatus(txSent.TxHash)
		if err := r.storage.UpdateTxSentStatus(txSent, status); err != nil {
			r.logger.WithFields(logrus.Fields{"function": "CheckTxSent() | UpdateTxSentStatus()"}).Errorln(err)
			return
		}
	}
}

func (r *BridgeSRV) handleTxSent(chain string, event *storage.Event, txType storage.TxType, backwardStatus storage.EventStatus,
	failedStatus storage.EventStatus) {
	txsSent := r.storage.GetTxsSentByType(chain, txType)
	if len(txsSent) == 0 {
		r.storage.UpdateEventStatus(event, backwardStatus)
		return
	}
	latestTx := txsSent[0]
	timeElapsed := time.Now().Unix() - latestTx.CreateTime
	autoRetryTimeout, autoRetryNum := r.getAutoRetryConfig(chain)
	txStatus := latestTx.Status

	if timeElapsed > autoRetryTimeout &&
		(txStatus == storage.TxSentStatusNotFound ||
			txStatus == storage.TxSentStatusInit ||
			txStatus == storage.TxSentStatusPending) {

		if len(txsSent) >= autoRetryNum {
			r.storage.UpdateEventStatus(event, failedStatus)
		} else {
			r.storage.UpdateEventStatus(event, backwardStatus)
		}
		r.storage.UpdateTxSentStatus(latestTx, storage.TxSentStatusLost)
	} else if txStatus == storage.TxSentStatusFailed {
		r.storage.UpdateEventStatus(event, failedStatus)
	}
}

// !!! TODO !!!

func (r *BridgeSRV) getAutoRetryConfig(chain string) (int64, int) {
	// if chain == "LA" {
	// 	autoRetryTimeout = r.Config.ChainConfig.BnbAutoRetryTimeout
	// 	autoRetryNum = r.Config.ChainConfig.BnbAutoRetryNum
	// } else {
	// 	autoRetryTimeout = r.Config.ChainConfig.WorkerChainAutoRetryTimeout
	// 	autoRetryNum = r.Config.ChainConfig.WorkerChainAutoRetryNum
	// }

	return 100000, 10
}
