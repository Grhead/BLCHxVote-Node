package Transport

import "VOX2/Blockchain"

type ChainHelp struct {
	Master string `form:"master" json:"master"`
	Count  int64  `form:"count" json:"count"`
}
type MasterHelp struct {
	Master string `form:"master" json:"master"`
}
type LastHashHelp struct {
	Hash string `form:"hash" json:"hash"`
}
type TransactionResponseHelp struct {
	TransactionStatus string `form:"status" json:"status"`
}
type UserHelp struct {
	User string `form:"user" json:"user"`
}
type BlockHelp struct {
	Block   *Blockchain.Block `form:"block" json:"block"`
	Size    uint64            `form:"size" json:"size"`
	Address string            `form:"address" json:"address"`
}
type TransactionHelp struct {
	Master   string           `form:"master" json:"master"`
	Sender   *Blockchain.User `form:"sender" json:"sender"`
	Receiver *ObjectHelp      `form:"receiver" json:"receiver"`
	Count    int64            `form:"count" json:"count"`
}
type ObjectHelp struct {
	PublicKey         string `json:"publicKey" gorm:"PublicKey"`
	VotingAffiliation string `json:"votingAffiliation" gorm:"VotingAffiliation"`
}

type BalanceHelp struct {
	Balance string `form:"balance" json:"balance"`
}
type SizeHelp struct {
	ChainSize string `form:"chainSize" json:"chainSize"`
}
type CheckHelp struct {
	AddTxStatus string `form:"addTxStatus" json:"addTxStatus"`
}
type CreateHelp struct {
	Status string `form:"genesisHash" json:"genesisHash"`
}
type VoterHelp struct {
	Pass string `form:"pass" json:"pass"`
}
