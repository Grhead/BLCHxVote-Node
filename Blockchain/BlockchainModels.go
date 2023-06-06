package Blockchain

import (
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Chain struct {
	Id    string
	Hash  string
	Block string
}

type Transaction struct {
	RandBytes []byte `form:"randBytes" json:"randBytes"`
	PrevBlock string `form:"prevBlock" json:"prevBlock"`
	Sender    string `form:"sender" json:"sender"`
	Receiver  string `form:"receiver" json:"receiver"`
	Value     int64  `form:"value" json:"value"`
	Signature string `form:"signature" json:"signature"`
	CurrHash  string `form:"currHash" json:"currHash"`
}

type Block struct {
	CurrHash     string                 `json:"currHash"`
	PrevHash     string                 `json:"prevHash"`
	TimeStamp    *timestamppb.Timestamp `json:"timeStamp"`
	Transactions []Transaction          `json:"transactions"`
	BalanceMap   map[string]int64       `json:"balanceMap"`
	Nonce        uint64                 `json:"nonce"`
	Difficulty   int64                  `json:"difficulty"`
	ChainMaster  string                 `json:"chainMaster"`
}

type User struct {
	Id                string `json:"uuid"`
	PublicKey         string `json:"publicKey"`
	IsUsed            bool   `json:"isUsed"`
	VotingAffiliation string `json:"affiliation"`
}

type ElectionSubjects struct {
	Id                string `json:"uuid" gorm:"Id"`
	PublicKey         string `json:"publicKey" gorm:"PublicKey"`
	Description       string `json:"description" gorm:"Description"`
	VotingAffiliation string `json:"votingAffiliation" gorm:"VotingAffiliation"`
}
