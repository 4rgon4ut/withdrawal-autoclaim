package main

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"log"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/withdrawal-autoclaim/bindings"
)

func run(client *ethclient.Client) error {
	headers := make(chan *types.Header)
	sub, err := client.SubscribeNewHead(context.Background(), headers)
	if err != nil {
		return fmt.Errorf("can't sunscribe to new blocks: %w", err)
	}

	for {
		select {

		case err := <-sub.Err():
			return fmt.Errorf("header sub error: %w", err)

		case header := <-headers:
			if err := accumulate(client, header); err != nil {
				return err
			}

		case <-time.Tick(time.Hour):
			batches := len(withdrawals) / BatchSize

			for i := 0; i <= batches; i++ {
				var list []common.Address

				if len(withdrawals) < BatchSize {
					list = withdrawals
					withdrawals = make([]common.Address, 0)
				} else {
					list = withdrawals[:BatchSize]
					withdrawals = withdrawals[BatchSize:]
				}

				if err := claim(client, list); err != nil {
					return fmt.Errorf("can't claim withdrawals: %w", err)
				}
			}
		}
	}
}

var withdrawals = make([]common.Address, 0)

func accumulate(client *ethclient.Client, header *types.Header) error {
	block, err := client.BlockByHash(context.Background(), header.Hash())
	if err != nil {
		return fmt.Errorf("can't get block by hash: %s err: %w", header.Hash(), err)
	}
	for _, w := range block.Withdrawals() {
		withdrawals = append(withdrawals, w.Address)
	}
	return nil
}

func claim(client *ethclient.Client, addrs []common.Address) error {
	opts, err := txOpts(client)
	if err != nil {
		return fmt.Errorf("transaction options build error: %w", err)
	}
	contract, err := bindings.NewDeposit(DepositContractAddress, client)
	if err != nil {
		return fmt.Errorf("can't create deposit contract binding: %w", err)
	}

	tx, err := contract.ClaimWithdrawals(opts, addrs)
	if err != nil {
		return fmt.Errorf("claim tx error: %w", err)
	}
	_ = tx
	// TODO: logging tx hash? wait for tx to execute?
	return nil
}

func txOpts(client *ethclient.Client) (*bind.TransactOpts, error) {
	privateKey, err := crypto.HexToECDSA(PrivateKey)
	if err != nil {
		log.Fatal(err)
	}

	publicKey := privateKey.Public()
	publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("can't cast public key to ECDSA")
	}

	fromAddress := crypto.PubkeyToAddress(*publicKeyECDSA)
	nonce, err := client.PendingNonceAt(context.Background(), fromAddress)
	if err != nil {
		return nil, fmt.Errorf("can't get account nonce: %w", err)
	}
	gasPrice, err := client.SuggestGasPrice(context.Background())
	if err != nil {
		return nil, fmt.Errorf("can't get gas price: %w", err)
	}
	chainID, err := client.ChainID(context.Background())
	if err != nil {
		return nil, fmt.Errorf("can't get chain id: %w", err)
	}

	auth, err := bind.NewKeyedTransactorWithChainID(privateKey, chainID)
	if err != nil {
		return nil, fmt.Errorf("can't create transactor: %w", err)
	}
	auth.Nonce = big.NewInt(int64(nonce))
	auth.Value = big.NewInt(0)     // in wei
	auth.GasLimit = uint64(300000) // in units
	auth.GasPrice = gasPrice

	return auth, nil
}

// func lastCallBlock(client *ethclient.Client) (*big.Int, error) {
// 	latest, err := client.HeaderByNumber(context.Background(), nil)
// 	if err != nil {
// 		return nil, fmt.Errorf("can't get latest block: %w", err)
// 	}
// 	num := latest.Number.Uint64()

// 	token, err := bindings.NewToken(TokenContractAddress, client)
// 	if err != nil {
// 		return nil, fmt.Errorf("can't bind token contract: %w", err)
// 	}
// 	eventsIterator, err := token.FilterTransfer(
// 		&bind.FilterOpts{
// 			Start: uint64(shapellaBlock),
// 			End:   &num,
// 		},
// 		[]common.Address{DepositContractAddress},
// 		[]common.Address{},
// 	)
// 	if err != nil {
// 		return nil, fmt.Errorf("can't filter transfers: %w", err)
// 	}

// 	for eventsIterator.Next() {
// 		addr := eventsIterator.Event.
// 		amount := eventsIterator.Event.Value
// 		if _, ok := transfersList[addr]; ok {
// 			transfersList[addr].Add(transfersList[addr], amount)
// 		} else {
// 			transfersList[addr] = amount
// 		}
// 	}

// }
