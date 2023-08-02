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

	lastSynced, err = readLastSynced()
	if err != nil {
		log.Fatal(err)
	}

	withdrawals = make(map[common.Address]struct{}, 0)
}

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
		for range time.Tick(time.Minute * 1) {
			log.Warn("lastSynced: ", lastSynced)
			head, err := client.HeaderByNumber(ctx, nil)
			if err != nil || head == nil {
				log.Errorf("can't get head: %s", err.Error())
				continue
			}
			go syncToHeadParallel(ctx, client, lastSynced, head.Number, retryQ)
		}
	}(ctx)

	// claims collected withdrawals every hour
	for range time.Tick(time.Minute * 60) {
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
		mux.Unlock()
	}
}

func main() {
	client, err := ethclient.Dial(rpcUrl)
	if err != nil {
		log.Fatal(err)
	}
	// parent context, all goroutines will stop on stop() call
	ctx, stop := context.WithCancel(context.Background())
	go Run(ctx, client)

	c := make(chan os.Signal, 1)
	// We'll accept graceful shutdowns when quit via SIGINT (Ctrl+C)
	// SIGKILL, SIGQUIT or SIGTERM (Ctrl+/) will not be caught.
	signal.Notify(c, os.Interrupt)

	// Block until we receive our signal
	<-c
	stop()
}
