package main

import (
	"log"
	"os"
	"strconv"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
)

const rpcUrl = "https://mainnet.infura.io"

var (
	shapellaBlock uint64 = 10379264

	BatchSize int

	DepositContractAddress common.Address
	TokenContractAddress   common.Address

	PrivateKey string
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
		BatchSize = 100
	}
}

func main() {
	client, err := ethclient.Dial(rpcUrl)
	if err != nil {
		log.Fatal(err)
	}

	if err := run(client); err != nil {
		log.Fatal(err)
	}
}
