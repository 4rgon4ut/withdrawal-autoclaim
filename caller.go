package main

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	log "github.com/sirupsen/logrus"
	"github.com/withdrawal-autoclaim/bindings"
)

func claimBatches(client *ethclient.Client) error {
	withdrawalAddrs := withdrawals.toSlice()
	batches := len(withdrawalAddrs) / BatchSize

	for i := 0; i <= batches; i++ {
		var list []common.Address

		if len(withdrawalAddrs) < BatchSize {
			list = withdrawalAddrs
		} else {
			list = withdrawalAddrs[:BatchSize]
			withdrawalAddrs = withdrawalAddrs[BatchSize:]
		}
		if err := claim(client, list); err != nil {
			return fmt.Errorf("can't claim withdrawals: %w", err)
		}
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
	log.Infof("sent claim tx hash: %s", tx.Hash())
	r, err := bind.WaitMined(context.Background(), client, tx)
	if err != nil {
		return fmt.Errorf("can't check if tx mined: %w", err)
	}
	log.Infof("tx mined on block: %d", r.BlockNumber.Uint64())
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
	auth.Value = big.NewInt(0) // in wei
	auth.GasLimit = uint64(0)  // in units
	auth.GasPrice = gasPrice

	return auth, nil
}
