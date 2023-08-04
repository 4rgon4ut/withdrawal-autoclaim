package main

import (
	"context"
	"math/big"
	"os"
	"os/signal"
	"strconv"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	log "github.com/sirupsen/logrus"
)

const rpcUrl = "https://rpc.gnosischain.com"

var (
	lastSynced uint64

	BatchSize int

	DepositContractAddress common.Address
	TokenContractAddress   common.Address

	PrivateKey string

	withdrawals Withdrawals

	metrics *Metrics
)

func init() {
	var err error

	DepositContractAddress = common.HexToAddress(os.Getenv("DEPOSIT_CONTRACT_ADDRESS"))
	if DepositContractAddress == common.HexToAddress("") {
		log.Fatal("no DEPOSIT_CONTRACT_ADDRESS provided")
	}
	TokenContractAddress = common.HexToAddress(os.Getenv("TOKEN_CONTRACT_ADDRESS"))
	if TokenContractAddress == common.HexToAddress("") {
		log.Fatal("no TOKEN_CONTRACT_ADDRESS provided")
	}
	PrivateKey = os.Getenv("PRIVATE_KEY")
	if PrivateKey == "" {
		log.Fatal("no PRIVATE_KEY provided")
	}
	BatchSize, err = strconv.Atoi(os.Getenv("BATCH_SIZE"))
	if err != nil {
		log.Fatal("invalid batch size value provided")
	}
	if BatchSize == 0 {
		BatchSize = 1000
	}
	lastSynced, err = getLastSynced()
	if err != nil {
		log.Fatal("no last synced block number provided")
	}

	withdrawals = make(map[common.Address]struct{}, 0)
	metrics = NewMetrics()
}

/*
Run is the main function of the application, it starts on the application start and runs
endlessly untill application stops.

Execution may be described in a few steps:

 1. Retry queue goroutine starts in parallel to retry every failed public RPC calls.
    Every bad response from public RPC being pushed to retry queue channel to reexecute the call.

 2. Syncs to the current head from provided <last_synced> block argument (on application start) collecting all withdrawals.

 3. Additional sync goroutine starts in parallel to sync new blocks every minute.

 4. Claim loop starts to claim withdrawals batches every <provided_time>.

    Every goroutine shares parent context and will stop onse parent context is canceled.
*/
func Run(ctx context.Context, client *ethclient.Client) {
	head, err := client.HeaderByNumber(ctx, nil)
	if err != nil {
		log.Fatalf("can't get chain head: %s", err.Error())
	}
	retryQ := make(chan uint64)
	// this function retry every failed block request
	go func(ctx context.Context) {
		for {
			select {
			case b := <-retryQ:
				metrics.rpcErrors.Add(float64(1))
				go processBlock(client, big.NewInt(int64(b)), retryQ)
			case <-ctx.Done():
				return
			}
		}
	}(ctx)
	// sync withdrawals list from lastSynced to current head block
	// needs to sync on application start
	if head.Number.Uint64() != lastSynced {
		syncToHeadParallel(ctx, client, lastSynced, head.Number, retryQ)
	}

	// goroutine that syncs new blocks every minute
	// shares parent context and will stop on parent context cancel function call on main
	go func(ctx context.Context) {
		for {
			select {
			case <-time.Tick(time.Minute * 1):
				head, err := client.HeaderByNumber(ctx, nil)
				if err != nil || head == nil {
					log.Errorf("can't get head: %s", err.Error())
					continue
				}
				syncToHeadParallel(ctx, client, lastSynced, head.Number, retryQ)
			case <-ctx.Done():
				return
			}
		}
	}(ctx)

	// claims collected withdrawals every hour
	for {
		select {
		// TODO: make time configurable ?
		case <-time.Tick(24 * time.Hour):
			// lock here to prevent races and make interaction with withdrawals list threadsafe
			// i.e. no other goroutines able to add new withdrawals to the list during claim and flushing
			mux.Lock()
			if err := claimBatches(client); err != nil {
				log.Errorf("can't claim batches: %s", err.Error())
				mux.Unlock()
				continue
			}
			// flush collected withdrawal accounts list
			withdrawals = make(map[common.Address]struct{}, 0)

			if err := writeLastSynced(lastSynced); err != nil {
				mux.Unlock()
				continue
			}
			//
			metrics.Commit()
			mux.Unlock()

		case <-ctx.Done():
			return
		}
	}
}

func main() {
	client, err := ethclient.Dial(rpcUrl)
	if err != nil {
		log.Fatal(err)
	}
	// parent context, all goroutines will stop on stop() call
	ctx, stop := context.WithCancel(context.Background())

	go metrics.serve(ctx)
	go Run(ctx, client)

	c := make(chan os.Signal, 1)
	// We'll accept graceful shutdowns when quit via SIGINT (Ctrl+C)
	// SIGKILL, SIGQUIT or SIGTERM (Ctrl+/) will not be caught.
	signal.Notify(c, os.Interrupt)

	// Block until receive SIGINT (kill -2)
	<-c
	stop()
}
