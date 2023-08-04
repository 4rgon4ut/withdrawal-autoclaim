package main

import (
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
	b := []byte(fmt.Sprintf("%d", lastSynced))
	err := os.WriteFile("./checkpoint/last_synced.txt", b, 0644)
	return err
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
