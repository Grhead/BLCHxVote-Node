package main

import (
	"VOX2/Blockchain"
	"VOX2/Transport"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"github.com/imroc/req/v3"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"log"
	"sort"
	"strconv"
	"strings"
)

type resultStruct struct {
	AddTxStatus string
}

func goAddBlock(block *Transport.BlockHelp, result resultStruct, goAddr string) {
	client := req.C().DevMode()
	_, err := client.R().
		SetBody(&block).
		SetSuccessResult(&result).
		Post(fmt.Sprintf("http://%s/addblock", strings.Trim(goAddr, "\"")))
	if err != nil && !strings.Contains(err.Error(), "No connection could be made because the target machine actively refused it.") {
		log.Fatal(err)
	}
}
func goAddTransaction() {
	Mutex.Lock()
	goroutineBlock := BlockForTransaction
	IsMining = true
	Mutex.Unlock()
	res := goroutineBlock.Accept(BreakMining)
	Mutex.Lock()
	IsMining = false
	if res == nil && strings.Compare(goroutineBlock.PrevHash, BlockForTransaction.PrevHash) == 0 {
		err := Blockchain.AddBlock(goroutineBlock)
		if err != nil {
			log.Fatal(err)
		}
		size, err := Blockchain.Size(goroutineBlock.ChainMaster)
		if err != nil {
			log.Fatal(err)
		}
		help := Transport.BlockHelp{
			Block:   goroutineBlock,
			Address: ThisServe,
			Size:    size,
		}
		err = pushBlockToNet(&help)
		if err != nil {
			log.Fatal(err)
		}
	}
	hash, err := Blockchain.LastHash("Start")
	if err != nil {
		log.Fatal(err)
	}
	BlockForTransaction, err = Blockchain.NewBlock(hash, "Start")
	if err != nil {
		log.Fatal(err)
	}
	Mutex.Unlock()
}
func goCompare(address string) {
	err := compareChains(address)
	if err != nil {
		log.Fatal(err)
	}
}

func pushBlockToNet(block *Transport.BlockHelp) error {
	//err := Blockchain.AddBlock(block.Block)
	//if err != nil {
	//	log.Fatal(err)
	//}
	var result resultStruct
	var returnErr error
	for _, addr := range OtherAddresses {
		goAddr := addr.String()
		go goAddBlock(block, result, goAddr)
	}
	if returnErr != nil {
		return returnErr
	}
	return nil
}

func AddBlock(pack *Transport.BlockHelp) (string, error) {
	block := pack.Block
	currSize, err := Blockchain.Size(block.ChainMaster)
	if err != nil {
		return "", err
	}
	num := pack.Size
	if currSize < num {
		fmt.Println("inside")
		go goCompare(pack.Address)
		return "ok", nil
	}
	Mutex.Lock()
	//err = Blockchain.AddBlock(block)
	//if err != nil {
	//	return "", err
	//}
	Mutex.Unlock()
	if IsMining {
		BreakMining <- true
		IsMining = false
	}
	hash, err := Blockchain.LastHash("Start")
	if err != nil {
		return "", nil
	}
	BlockForTransaction, err = Blockchain.NewBlock(hash, "Start")
	if err != nil {
		return "", nil
	}
	return "ok", nil
}

func AddTransaction(BlockTx *Transport.TransactionHelp) (string, error) {
	var err error
	var tx *Blockchain.Transaction
	if BlockTx.Sender == nil {
		log.Println(BlockTx)
		tx, err = Blockchain.NewTransactionFromChain(BlockTx.Master, BlockTx.Receiver, BlockTx.Count)
		if err != nil {
			return "", err
		}
	} else {
		hash, errLastHash := Blockchain.LastHash(BlockTx.Master)
		if errLastHash != nil {
			return "", errLastHash
		}
		tx, err = Blockchain.NewTransaction(BlockTx.Sender, BlockTx.Receiver, hash, BlockTx.Count)
		if err != nil {
			return "", err
		}
	}
	if len(BlockForTransaction.Transactions) == Blockchain.TxsLimit {
		return "", errors.New("transactions limit in blocks")
	}
	Mutex.Lock()
	BlockForTransaction.ChainMaster = BlockTx.Master
	err = BlockForTransaction.AddTransaction(tx)
	if err != nil {
		return "", err
	}
	Mutex.Unlock()
	fmt.Println(len(BlockForTransaction.Transactions))
	//if len(BlockForTransaction.Transactions) == Blockchain.TxsLimit {
	go goAddTransaction()
	//}
	return "ok", nil
}

func compareChains(address string) error {
	dbNode, err := gorm.Open(sqlite.Open("Database/NodeDb.db"), &gorm.Config{})
	if err != nil {
		return err
	}
	var blocksResponse []*Blockchain.Block
	client := req.C().DevMode()
	_, err = client.R().SetSuccessResult(&blocksResponse).
		Get(fmt.Sprintf("http://%s/getdb", strings.Trim(address, "\"")))
	if err != nil {
		return err
	}
	someGenesis := blocksResponse[0]
	if strings.Compare(someGenesis.CurrHash, someGenesis.Hash()) != 0 {
		return errors.New("hashes are not the same")
	}
	var arrayToMerge []*Blockchain.Chain
	serializeGenesisBlock, err := Blockchain.SerializeBlock(someGenesis)
	if err != nil {
		return err
	}
	genesisBlock := Blockchain.Chain{
		Id:    uuid.NewString(),
		Hash:  someGenesis.CurrHash,
		Block: serializeGenesisBlock,
	}
	arrayToMerge = append(arrayToMerge, &genesisBlock)
	for _, v := range blocksResponse {
		if v != blocksResponse[0] {
			if v == nil {
				return errors.New("block is nil")
			}
			serializeBlock, errSerialize := Blockchain.SerializeBlock(v)
			if errSerialize != nil {
				return errSerialize
			}
			block := Blockchain.Chain{
				Id:    uuid.NewString(),
				Hash:  v.CurrHash,
				Block: serializeBlock,
			}
			arrayToMerge = append(arrayToMerge, &block)
		}
	}
	Mutex.Lock()
	errDelete := dbNode.Exec("DELETE FROM Chains WHERE Id != 0")
	if errDelete.Error != nil {
		return errDelete.Error
	}
	for _, v := range arrayToMerge {
		errInsert := dbNode.Exec("INSERT INTO Chains (Id, Hash, Block) VALUES ($1, $2, $3)",
			uuid.NewString(),
			v.Hash,
			v.Block,
		)
		if errInsert.Error != nil {
			return errInsert.Error
		}
	}
	hash, err := Blockchain.LastHash("Start")
	if err != nil {
		log.Fatal(err)
	}
	BlockForTransaction, err = Blockchain.NewBlock(hash, "Start")
	if err != nil {
		log.Fatal(err)
	}
	Mutex.Unlock()
	if IsMining {
		BreakMining <- true
		IsMining = false
	}
	return nil
}

func NewChain(chainMaster string, count int64) (string, error) {
	genesis, err := Blockchain.NewChain(count, chainMaster)
	if err != nil {
		log.Println(err)
		return "", err
	}
	return genesis.CurrHash, nil
}

func GetBlocks(pack *Transport.MasterHelp) ([]*Blockchain.Block, error) {
	blocks, err := Blockchain.GetFullChain(pack.Master)
	if err != nil {
		return nil, err
	}
	sort.Slice(blocks, func(i, j int) bool {
		return blocks[i].TimeStamp.AsTime().After(blocks[j].TimeStamp.AsTime())
	})
	return blocks, nil
}

func GetLastHash(pack *Transport.MasterHelp) (string, error) {
	return Blockchain.LastHash(pack.Master)
}

func GetBalance(pack *Transport.UserHelp) (string, error) {
	balance, err := Blockchain.Balance(pack.User)
	if err != nil {
		return "", err
	}
	return strconv.FormatInt(balance, 10), nil
}

func GetChainSize(pack *Transport.MasterHelp) (string, error) {
	size, err := Blockchain.Size(pack.Master)
	if err != nil {
		return "", err
	}
	return strconv.FormatUint(size, 10), nil
}
