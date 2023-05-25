package Basic

import (
	"VOX2/Blockchain"
	"VOX2/Transport"
	"errors"
	"fmt"
	"github.com/imroc/req/v3"
	"github.com/valyala/fastjson"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"os"
	"reflect"
	"strconv"
	"strings"
)

func CallCreateVoters(voter interface{}, master string) ([]*Blockchain.User, error) {
	var resultItems []*Blockchain.User
	switch voter.(type) {
	case int:
		max, err := strconv.Atoi(fmt.Sprintf("%v", voter))
		if err != nil {
			return nil, err
		}
		for i := 0; i < max; i++ {
			item, errNewPublicKey := Blockchain.NewPublicKeyItem(master)
			if errNewPublicKey != nil {
				return nil, errNewPublicKey
			}
			resultItems = append(resultItems, item)
		}
	case string:
		err := Blockchain.NewDormantUser(fmt.Sprintf("%v", voter))
		if err != nil {
			return nil, err
		}
		item, err := Blockchain.NewPublicKeyItem(master)
		if err != nil {
			return nil, err
		}
		resultItems = append(resultItems, item)
	default:
		return nil, errors.New("invalid type")
	}
	return resultItems, nil
}

func CallViewCandidates() ([]*Blockchain.ElectionSubjects, error) {
	db, err := gorm.Open(sqlite.Open("Database/ContractDB.db"), &gorm.Config{})
	if err != nil {
		return nil, err
	}
	var Candidates []*Blockchain.ElectionSubjects
	db.Table("ElectionSubjects").Find(&Candidates)
	return Candidates, nil
}

func CallNewCandidate(description string, affiliation string) (*Blockchain.ElectionSubjects, error) {
	candidate, err := Blockchain.NewCandidate(description, affiliation)
	if err != nil {
		return nil, err
	}
	return candidate, nil
}

func GetBalance(userAddress string) (*Transport.BalanceHelp, error) {
	addresses, err := readAddresses()
	if err != nil {
		return nil, err
	}
	userAddressStruct := Transport.UserHelp{
		User: userAddress,
	}
	var userBalance Transport.BalanceHelp
	client := req.C().DevMode()
	for _, addr := range addresses {
		resp, errReq := client.R().
			SetBody(&userAddressStruct).
			SetSuccessResult(&userBalance).
			Post(fmt.Sprintf("http://%s/getbalance", strings.Trim(addr.String(), "\"")))
		if errReq != nil && !strings.Contains(errReq.Error(), "No connection could be made because the target machine actively refused it.") {
			return nil, errReq
		}
		if errReq == nil {
			if resp.Body == nil {
				continue
			}
		}
	}
	return &userBalance, nil
}

func ChainSize(master string) (string, error) {
	addresses, err := readAddresses()
	if err != nil {
		return "", err
	}
	masterChain := Transport.MasterHelp{Master: master}
	var chainSize Transport.SizeHelp
	client := req.C().DevMode()
	resp, err := client.R().
		SetBody(&masterChain).
		SetSuccessResult(&chainSize).
		Post(fmt.Sprintf("http://%s/getchainsize", strings.Trim(addresses[0].String(), "\"")))
	if err != nil {
		return "", err
	}
	if resp.Body == nil {
		return "", errors.New("empty response")
	}
	return chainSize.ChainSize, nil
}

func GetPartOfChain(master string) ([]*Blockchain.Block, error) {
	addresses, err := readAddresses()
	if err != nil {
		return nil, err
	}
	var partOfChain []*Blockchain.Block
	ChainMaster := Transport.MasterHelp{
		Master: master,
	}
	client := req.C().DevMode()
	resp, err := client.R().
		SetBody(&ChainMaster).
		SetSuccessResult(&partOfChain).
		Post(fmt.Sprintf("http://%s/getblock", strings.Trim(addresses[0].String(), "\"")))
	if err != nil {
		return nil, err
	}
	if resp.Body == nil {
		return nil, errors.New("empty response")
	}
	return partOfChain, nil
}

func GetFullChain() ([]*Blockchain.Block, error) {
	addresses, err := readAddresses()
	if err != nil {
		return nil, err
	}
	var fullChain []*Blockchain.Block
	client := req.C().DevMode()
	resp, err := client.R().SetSuccessResult(&fullChain).
		Get(fmt.Sprintf("http://%s/getdb", strings.Trim(addresses[0].String(), "\"")))
	if err != nil {
		return nil, err
	}
	if resp.Body == nil {
		return nil, errors.New("empty response")
	}
	return fullChain, nil
}

func AcceptNewUser(Pass string, salt string, PublicKey string) (string, error) {
	//TODO add time verification
	//t, _ := time.ParseDuration(EndTime)
	//t1, _ := time.ParseDuration(LimitTime())
	//if t1 > t {
	//	return "time"
	//}
	private, err := Blockchain.RegisterGeneratePrivate(Pass, salt, PublicKey)
	if err != nil {
		return "", err
	}
	return private, nil
}

func AcceptLoadUser(PublicK string, PrivateK string) (*Blockchain.User, error) {
	//TODO add time verification
	//t, _ := time.ParseDuration(EndTime)
	//t1, _ := time.ParseDuration(LimitTime())
	//if t1 > t {
	//	return "2"
	//}
	UserPrivate, err := Blockchain.LoadToEnterAlreadyUserPrivate(PrivateK)
	if err != nil {
		return nil, err
	}
	UserPublic, err := Blockchain.LoadToEnterAlreadyUserPublic(PublicK)
	if err != nil {
		return nil, err
	}
	if UserPublic.PublicKey != PublicK || UserPrivate.PublicKey != PublicK || !reflect.DeepEqual(UserPrivate, UserPublic) {
		return nil, errors.New("invalid input")
	}
	bal, err := GetBalance(UserPublic.Address())
	if err != nil {
		return nil, err
	}
	fmt.Println(bal)
	if bal == nil {
		return nil, errors.New("zero balance")
	}
	return UserPublic, nil
}

func ChainTXBlock(receiver string, master string, num uint64) (string, error) {
	addresses, err := readAddresses()
	if err != nil {
		return "", err
	}
	input := Transport.MasterHelp{
		Master: master,
	}
	var lastHash Transport.LastHashHelp
	var txStatus Transport.TransactionResponseHelp
	for _, addr := range addresses {
		client := req.C().DevMode()
		resp, errReq := client.R().
			SetBody(&input).
			SetSuccessResult(&lastHash).
			Post(fmt.Sprintf("http://%s/getlasthash", strings.Trim(addr.String(), "\"")))
		if errReq != nil && !strings.Contains(errReq.Error(), "No connection could be made because the target machine actively refused it.") {
			return "", errReq
		}
		if errReq == nil {
			if resp.Body == nil {
				continue
			}
		}
		balance, errBalance := GetBalance(master)
		if errBalance != nil {
			return "", errBalance
		}
		chainBalance, errConversion := strconv.Atoi(balance.Balance)
		if errConversion != nil {
			return "", errConversion
		}
		if uint64(chainBalance) < num {
			return "", errors.New("not enough chain founds")
		}
		public, errLoad := Blockchain.LoadToEnterAlreadyUserPublic(receiver)
		if errLoad != nil {
			return "", errLoad
		}
		tx, errNewTx := Blockchain.NewTransactionFromChain(master, public, num)
		transactionToNet := Transport.TransactionHelp{
			Master: master,
			Tx:     tx,
		}
		if errNewTx != nil {
			return "", errNewTx
		}
		resp, errReq = client.R().
			SetBody(&transactionToNet).
			SetSuccessResult(&txStatus).
			Post(fmt.Sprintf("http://%s/addtx", strings.Trim(addr.String(), "\"")))
		if errReq != nil && !strings.Contains(errReq.Error(), "No connection could be made because the target machine actively refused it.") {
			return "", errReq
		}
		if errReq == nil {
			if resp.Body == nil {
				continue
			}
		}
	}
	return txStatus.TransactionStatus, nil
}

func readAddresses() ([]*fastjson.Value, error) {
	file, err := os.ReadFile("LowConf/addr.json")
	if err != nil {
		return nil, err
	}
	var p fastjson.Parser
	v, err := p.Parse(string(file))
	if err != nil {
		return nil, err
	}
	return v.GetArray("addresses"), nil
}

//addresses, err := readAddresses()
//if err != nil {
//return "", err
//}

//client := req.C().DevMode()
//_, err := client.R().
//SetBody(&block).
//SetSuccessResult(&result).
//Post(fmt.Sprintf("http://%s/addblock", strings.Trim(goAddr, "\"")))
