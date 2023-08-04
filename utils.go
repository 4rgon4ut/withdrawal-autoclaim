package main

import (
	"flag"
	"fmt"
	"os"
	"strconv"

	"github.com/ethereum/go-ethereum/common"
)

// TODO: rename to addresses ?
type Withdrawals map[common.Address]struct{}

// toSlice return slice of unique withdrawal addresses
func (w Withdrawals) toSlice() []common.Address {
	addrs := make([]common.Address, 0)
	for a := range w {
		addrs = append(addrs, a)
	}
	return addrs
}

// writeLastSynced write lastSynced block to last_synced.txt
func writeLastSynced(lastSynced uint64) error {
	f, err := os.OpenFile("./checkpoint/last_synced.txt", os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("can't open last_synced.txt: %w", err)
	}
	defer f.Close()
	if _, err := f.WriteString(fmt.Sprintf("%d", lastSynced)); err != nil {
		return fmt.Errorf("can't write new checkpoint to last_synced.txt: %w", err)
	}
	return nil
}

// readLastSynced return lastSynced block from last_synced.txt
func readLastSynced() (uint64, error) {
	content, err := os.ReadFile("./checkpoint/last_synced.txt")
	if err != nil {
		return 0, fmt.Errorf("can't read last_synced.txt: %w", err)
	}
	i, err := strconv.Atoi(string(content))
	if err != nil {
		return 0, fmt.Errorf("can't parse lastSynced number: %w", err)
	}
	return uint64(i), nil
}

func getLastSynced() (uint64, error) {
	var err error
	lastSyncedArg := flag.Int("last-synced", 0, "last synced block")
	flag.Parse()
	lastS := uint64(*lastSyncedArg)
	if lastS == 0 {
		lastS, err = readLastSynced()
		if err != nil {
			return 0, err
		}
		return lastS, nil
	}
	if err := writeLastSynced(lastS); err != nil {
		return 0, err
	}
	return lastS, nil
}
