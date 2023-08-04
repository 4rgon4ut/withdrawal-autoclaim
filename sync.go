package main

import (
	"context"
	"math/big"
	"sync"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	log "github.com/sirupsen/logrus"
)

var mux sync.Mutex
var wg sync.WaitGroup

// syncToHeadParallel spreads amount of blocks to sync between 3 goroutines.
// Function wait all spawned goroutines to finish.
func syncToHeadParallel(ctx context.Context, client *ethclient.Client, lastSynced uint64, head *big.Int, retryQ chan uint64) {
	var threads uint64 = 3

	diff := head.Uint64() - lastSynced
	portion := diff / threads
	for j := 0; j < int(threads); j++ {
		wg.Add(1)
		start := lastSynced + uint64(j)*portion
		go syncPortion(ctx, client, start, start+portion, retryQ)
	}

	wg.Wait()
}

// syncPortion calls block processing for every block from start to end.
func syncPortion(ctx context.Context, client *ethclient.Client, start, end uint64, retryQ chan uint64) {
	defer wg.Done()
	// TODO: cut logging ?
	log.Infof("syncing blocks %d --> %d", start, end)
	for i := start; i <= end; i++ {
		processBlock(client, big.NewInt(int64(i)), retryQ)
	}
}

// processBlock try to get block by number and accumulate withdrawals.
// If processing fails, block number being pushed to retryQ channel.
func processBlock(client *ethclient.Client, blockNum *big.Int, retryQ chan uint64) {
	block, err := client.BlockByNumber(context.Background(), blockNum)
	if err != nil {
		log.Error("get block by number error: %w", err)
		retryQ <- blockNum.Uint64()
		return
	}
	if block == nil {
		log.Warn("nil block response")
		retryQ <- blockNum.Uint64()
		return
	}

	if err := accumulate(client, block); err != nil {
		log.Error("accumulation error: %w", err)
		retryQ <- blockNum.Uint64()
	}
}

// accumulates all unique withdrawals addresses and increments lastSynced counter.
func accumulate(client *ethclient.Client, block *types.Block) error {
	mux.Lock()
	defer mux.Unlock()
	for _, w := range block.Withdrawals() {
		withdrawals[w.Address] = struct{}{}
		metrics.withdrawalsCounter++
	}
	if block.Number().Uint64() > lastSynced {
		lastSynced = block.Number().Uint64()
	}
	// TODO: cut excess logging ?
	log.Infof("synced block: %d", block.Number().Uint64())
	return nil
}
