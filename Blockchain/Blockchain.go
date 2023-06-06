package Blockchain

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"github.com/spf13/viper"
	"github.com/valyala/fastjson"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"io"
	"log"
	"net/http"
	"sort"
	"strconv"
)

const (
	TxsLimit = 4
)

func init() {
	viper.SetConfigFile("./LowConf/config.env")
	err := viper.ReadInConfig()
	if err != nil {
		fmt.Println(err)
		return
	}
}

func NewChain(VotesCount int64, ChainMaster string) (*Block, error) {
	db, err := gorm.Open(sqlite.Open("Database/NodeDb.db"), &gorm.Config{})
	if err != nil {
		return nil, err
	}
	curTime, errGetTime := GetTime()
	if errGetTime != nil {
		return nil, errGetTime
	}
	var blocks []*Chain
	db.Find(&blocks)
	for _, v := range blocks {
		desBlock, errDes := DeserializeBlock(v.Block)
		if errDes != nil {
			return nil, errDes
		}
		if desBlock.ChainMaster == ChainMaster {
			return nil, errors.New("affiliation already exist")
		}
	}

	genesis := &Block{
		PrevHash:    HashSum(ChainMaster),
		BalanceMap:  make(map[string]int64),
		TimeStamp:   curTime,
		ChainMaster: ChainMaster,
	}
	genesis.BalanceMap[ChainMaster] = VotesCount
	genesis.CurrHash = genesis.Hash()
	err = AddBlock(genesis)
	if err != nil {
		return nil, err
	}
	return genesis, nil
}

func NewBlock(prevHash string, chainMaster string) (*Block, error) {
	VarConf := viper.GetString("DIFFICULTY")
	difficulty, err := strconv.Atoi(VarConf)
	if err != nil {
		return nil, err
	}
	return &Block{
		Difficulty:  int64(difficulty),
		PrevHash:    prevHash,
		BalanceMap:  make(map[string]int64),
		ChainMaster: chainMaster,
	}, nil
}

func NewTransaction(
	fromUser *User,
	toUser *User,
	lastHash string,
	value int64) (*Transaction, error) {
	VarConf := viper.GetString("RAND_BYTES")
	VarConfConversion, err := strconv.Atoi(VarConf)
	if err != nil {
		return nil, err
	}
	randBytes, err := GenerateRandomBytes(uint64(VarConfConversion))
	if err != nil {
		return nil, err
	}
	if fromUser.Affiliation != toUser.Affiliation {
		return nil, errors.New("affiliation does not match")
	}
	tran := &Transaction{
		RandBytes: randBytes,
		PrevBlock: lastHash,
		Sender:    fromUser.Address(),
		Receiver:  toUser.Address(),
		Value:     value,
	}
	tran.CurrHash = tran.Hash()
	tran.Signature = tran.Sign(fromUser.Address() + toUser.Address())
	return tran, err
}

func NewTransactionFromChain(
	master string,
	toUser *User,
	value int64) (*Transaction, error) {
	VarConf := viper.GetString("RAND_BYTES")
	VarConfConversion, err := strconv.Atoi(VarConf)
	if err != nil {
		return nil, err
	}
	randBytes, err := GenerateRandomBytes(uint64(VarConfConversion))
	if err != nil {
		return nil, err
	}
	lastHash, err := LastHash(master)
	if err != nil {
		return nil, err
	}
	if master != toUser.Affiliation {
		return nil, errors.New("affiliation does not match")
	}
	tran := &Transaction{
		RandBytes: randBytes,
		PrevBlock: lastHash,
		Sender:    master,
		Receiver:  toUser.Address(),
		Value:     value,
	}
	tran.CurrHash = tran.Hash()
	tran.Signature = tran.Sign(master + toUser.Address())
	return tran, err
}

func LastHash(master string) (string, error) {
	db, err := gorm.Open(sqlite.Open("Database/NodeDb.db"), &gorm.Config{})
	if err != nil {
		return "", err
	}
	var hash string
	var chain []*Chain
	var blocks []*Block
	db.Find(&chain)
	for _, v := range chain {
		deserializedBlock, errDes := DeserializeBlock(v.Block)
		if errDes != nil {
			return "", errDes
		}
		blocks = append(blocks, deserializedBlock)
	}
	sort.Slice(blocks, func(i, j int) bool {
		return blocks[i].TimeStamp.AsTime().After(blocks[j].TimeStamp.AsTime())
	})
	for _, v := range blocks {
		if v.ChainMaster == master {
			hash = v.CurrHash
			break
		}
	}
	return hash, nil
}

func AddBlock(block *Block) error {
	db, err := gorm.Open(sqlite.Open("Database/NodeDb.db"), &gorm.Config{})
	if err != nil {
		return err
	}
	serializedBlock, err := SerializeBlock(block)
	if err != nil {
		return err
	}
	errInsert := db.Exec("INSERT INTO Chains (Id, Hash, Block) VALUES ($1, $2, $3)",
		uuid.NewString(),
		block.CurrHash,
		serializedBlock,
	)
	if errInsert.Error != nil {
		return errInsert.Error
	}
	return nil
}

// NewDormantUser same with AddPass (BLCHxVote)
func NewDormantUser(identifier string) error {
	db, err := gorm.Open(sqlite.Open("Database/ContractDB.db"), &gorm.Config{})
	if err != nil {
		return err
	}
	privateGenKey, err := GenerateKey()
	if err != nil {
		return err
	}
	var isUsed string
	db.Raw("SELECT Id FROM RelationPatterns WHERE PersonIdentifier = $1",
		identifier).Scan(&isUsed)
	if isUsed != "" {
		return errors.New("identifier not allowed")
	}
	db.Exec("INSERT INTO RelationPatterns (Id, PersonIdentifier, PrivateKeyTemplate) VALUES ($1, $2, $3)",
		uuid.NewString(),
		HashSum(identifier),
		privateGenKey)
	return nil
}

func LoadToEnterAlreadyUserPrivate(privateKey string) (*User, error) {
	db, err := gorm.Open(sqlite.Open("Database/ContractDB.db"), &gorm.Config{})
	if err != nil {
		return nil, err
	}
	var publicKey string
	var LoadedUser *User
	db.Raw("SELECT PublicKey FROM KeyLinks WHERE PrivateKey = $1",
		privateKey).Scan(&publicKey)
	var affiliation string
	errWhere := db.Table("KeyLinks").Where("PublicKey = ?", publicKey).First(&LoadedUser)
	if errWhere.Error != nil {
		return nil, errWhere.Error
	}
	db.Raw("SELECT VotingAffiliation FROM PublicKeySets WHERE PublicKey = $1",
		LoadedUser.Address()).Scan(&affiliation)
	LoadedUser.Affiliation = affiliation
	return LoadedUser, nil
}

func LoadToEnterAlreadyUserPublic(publicKey string) (*User, error) {
	db, err := gorm.Open(sqlite.Open("Database/ContractDB.db"), &gorm.Config{})
	if err != nil {
		return nil, err
	}
	var LoadedUser *User
	errWhere := db.Table("KeyLinks").Where("PublicKey = ?", publicKey).First(&LoadedUser)
	if errWhere.Error != nil {
		return nil, errWhere.Error
	}
	var affiliation string
	db.Raw("SELECT VotingAffiliation FROM PublicKeySets WHERE PublicKey = $1",
		publicKey).Scan(&affiliation)
	LoadedUser.Affiliation = affiliation
	return LoadedUser, nil
}

// SelectByIdentifier Same with Purse (BLCHxVote) and RegisterGeneratePrivate
/*func SelectByIdentifier(identifier string) (string, error) {
	db, err := gorm.Open(sqlite.Open("Database/NodeDb.db"), &gorm.Config{})
	if err != nil {
		return "", err
	}
	var privateKeyTemplate string
	db.Raw("SELECT PrivateKeyTemplate FROM RelationPatterns WHERE PersonIdentifier = $1",
		SetHash(identifier)).Scan(&privateKeyTemplate)
	if privateKeyTemplate == "" {
		return "", errors.New("identifier does not exist")
	}
	return privateKeyTemplate, nil
}*/

// NewPublicKeyItem Same with NewUser(BLCHxVote)
func NewPublicKeyItem(affiliation string) (*User, error) {
	db, err := gorm.Open(sqlite.Open("Database/ContractDB.db"), &gorm.Config{})
	if err != nil {
		return nil, err
	}
	tempKey, err := GenerateKey()
	if err != nil {
		return nil, err
	}
	tempUUID := uuid.New()
	db.Exec("INSERT INTO PublicKeySets (Id, PublicKey, IsUsed, VotingAffiliation) VALUES ($1, $2, $3, $4)",
		tempUUID,
		tempKey,
		false,
		affiliation)
	return &User{
		Id:          tempUUID.String(),
		PublicKey:   tempKey,
		IsUsed:      false,
		Affiliation: affiliation,
	}, nil
}

func NewCandidate(description string, affiliation string) (*ElectionSubjects, error) {
	db, err := gorm.Open(sqlite.Open("Database/ContractDB.db"), &gorm.Config{})
	if err != nil {
		return nil, err
	}
	tempUUID := uuid.New()
	tempKey, errKeyGen := GenerateKey()
	if errKeyGen != nil {
		return nil, errKeyGen
	}
	db.Exec("INSERT INTO ElectionSubjects (Id, PublicKey,Description, VotingAffiliation) VALUES ($1, $2, $3, $4)",
		tempUUID,
		tempKey,
		description,
		affiliation)
	return &ElectionSubjects{
		Id:                tempUUID.String(),
		PublicKey:         tempKey,
		Description:       description,
		VotingAffiliation: affiliation,
	}, nil
}

func Size(master string) (uint64, error) {
	db, err := gorm.Open(sqlite.Open("Database/NodeDb.db"), &gorm.Config{})
	if err != nil {
		return 0, err
	}
	var index uint64
	var blocks []*Chain
	db.Find(&blocks)
	for _, v := range blocks {
		desBlock, err := DeserializeBlock(v.Block)
		if err != nil {
			return 0, err
		}
		if desBlock.ChainMaster == master {
			index++
		}
	}
	return index, nil
}

func Balance(moneyMan string) (int64, error) {
	db, err := gorm.Open(sqlite.Open("Database/NodeDb.db"), &gorm.Config{})
	if err != nil {
		return 0, err
	}
	var balance int64
	var blocks []*Chain
	db.Find(&blocks)
	for _, v := range blocks {
		desBlock, errDes := DeserializeBlock(v.Block)
		if errDes != nil {
			return 0, errDes
		}
		if value, ok := desBlock.BalanceMap[moneyMan]; ok {
			balance = value
			break
		}
	}
	return balance, nil
}

func RegisterGeneratePrivate(passport string, salt string, PublicKey string) (string, error) {
	db, err := gorm.Open(sqlite.Open("Database/ContractDB.db"), &gorm.Config{})
	if err != nil {
		return "", err
	}
	var checkPublicKey string
	db.Raw("SELECT Id FROM PublicKeySets WHERE PublicKey = $1",
		PublicKey).Scan(&checkPublicKey)
	if checkPublicKey == "" {
		return "", errors.New("public key is invalid")
	}
	var checkTemplate string
	db.Raw("SELECT PrivateKeyTemplate FROM RelationPatterns WHERE PersonIdentifier = $1",
		HashSum(passport)).Scan(&checkTemplate)
	if checkTemplate == "" {
		return "", errors.New("identifier does not exist")
	}
	var checkIsUsed bool
	db.Raw("SELECT isUsed FROM PublicKeySets WHERE PublicKey = $1",
		PublicKey).Scan(&checkIsUsed)
	if checkIsUsed == true {
		return "", errors.New("public key is already used")
	}
	var checkIsCandidate string
	db.Raw("SELECT Id FROM ElectionSubjects WHERE PublicKey = $1",
		PublicKey).Scan(&checkIsCandidate)
	if checkIsCandidate != "" {
		return "", errors.New("public key is not allowed")
	}
	errExec := db.Exec("UPDATE PublicKeySets SET isUsed = 1 WHERE PublicKey = $1", PublicKey)
	if errExec.Error != nil {
		return "", errExec.Error
	}
	hash := sha256.Sum256([]byte(checkTemplate + salt))
	err = ImportToDB(fmt.Sprintf("%x", hash[:]), PublicKey)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", hash[:]), nil
}

func GetVotingAffiliation(PublicKey string) (string, error) {
	db, err := gorm.Open(sqlite.Open("Database/ContractDB.db"), &gorm.Config{})
	if err != nil {
		return "", err
	}
	var checkVotingAffiliation string
	db.Raw("SELECT VotingAffiliation FROM PublicKeySets WHERE PublicKey = $1",
		PublicKey).Scan(&checkVotingAffiliation)
	return checkVotingAffiliation, nil
}

// GenerateKey TODO Rewrite
func GenerateKey() (string, error) {
	TimeUrl := viper.GetString("TIME_URL")
	resp, err := http.Get(TimeUrl)
	if err != nil {
		defer func(Body io.ReadCloser) {
			err := Body.Close()
			if err != nil {
				log.Fatal(err)
			}
		}(resp.Body)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	var p fastjson.Parser
	v, err := p.Parse(string(body))
	if err != nil {
		return "", err
	}
	hash := v.GetStringBytes("dateTime")
	return HashSum(string(hash)), nil
}

func GetFullChain(master string) ([]*Block, error) {
	db, err := gorm.Open(sqlite.Open("Database/NodeDb.db"), &gorm.Config{})
	if err != nil {
		return nil, err
	}
	var blocks []*Chain
	var resultMasterBlocks []*Block
	db.Find(&blocks)
	for _, v := range blocks {
		desBlock, errDes := DeserializeBlock(v.Block)
		if errDes != nil {
			return nil, errDes
		}
		if desBlock.ChainMaster == master {
			resultMasterBlocks = append(resultMasterBlocks, desBlock)
		}
	}
	return resultMasterBlocks, nil
}

func GetFullDb() ([]*Block, error) {
	db, err := gorm.Open(sqlite.Open("Database/NodeDb.db"), &gorm.Config{})
	if err != nil {
		return nil, err
	}
	var blocks []*Chain
	var resultMasterBlocks []*Block
	db.Find(&blocks)
	for _, v := range blocks {
		desBlock, err := DeserializeBlock(v.Block)
		if err != nil {
			return nil, err
		}
		resultMasterBlocks = append(resultMasterBlocks, desBlock)
	}
	return resultMasterBlocks, nil
}

//func DbSize() (uint64, error) {
//	db, err := gorm.Open(sqlite.Open("Database/NodeDb.db"), &gorm.Config{})
//	if err != nil {
//		return 0, err
//	}
//	var index uint64
//	var blocks []*Chain
//	db.Find(&blocks)
//	for range blocks {
//		if err != nil {
//			return 0, err
//		}
//		index++
//	}
//	return index, nil
//}

//func GetBlock(uuidR uuid.UUID) (uuid.UUID, *Block, error) {
//	db, err := gorm.Open(sqlite.Open("Database/NodeDb.db"), &gorm.Config{})
//	if err != nil {
//		return uuid.Nil, nil, err
//	}
//	var chainBlock *Chain
//	db.Where("Id = ?", uuidR).Find(&chainBlock)
//	desBlock, err := DeserializeBlock(chainBlock.Block)
//	if err != nil {
//		return uuid.Nil, nil, nil
//	}
//	return uuidR, desBlock, nil
//}
