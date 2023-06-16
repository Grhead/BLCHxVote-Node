package main

import (
	"VOX2/Blockchain"
	"VOX2/Transport"
	"errors"
	"github.com/gin-contrib/cors"
	"github.com/gin-contrib/timeout"
	"github.com/gin-gonic/gin"
	"github.com/valyala/fastjson"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

var Mutex sync.Mutex
var IsMining bool
var BreakMining = make(chan bool)
var ThisServe string
var OtherAddresses []*fastjson.Value
var BlockForTransaction *Blockchain.Block

func SetAddress(server string) {
	ThisServe = ":" + server
}

func init() {
	hash, err := Blockchain.LastHash("Start")
	if err != nil {
		log.Fatalln(err)
	}
	BlockForTransaction, err = Blockchain.NewBlock(hash, "Start")
	if err != nil {
		log.Fatalln(err)
	}
	OtherAddresses, err = ReadAddresses()
	if err != nil {
		log.Fatalln(err)
	}
	for i, k := range OtherAddresses {
		if strings.Contains(k.String(), ThisServe) {
			func(slice []*fastjson.Value, s int) []*fastjson.Value {
				return append(slice[:s], slice[s+1:]...)
			}(OtherAddresses, i)
		}
	}
}
func emptySuccessResponse(c *gin.Context) {
	time.Sleep(200 * time.Microsecond)
	c.String(http.StatusOK, "")
}

func main() {
	rootCmd.Flags().StringVarP(&option, "set", "s", "", "Set address")
	Execute()
	if ThisServe == "" {
		panic(errors.New("empty address"))
	}
	router := gin.Default()
	router.Use(cors.New(cors.Config{
		AllowAllOrigins:  true,
		AllowMethods:     []string{"POST", "GET"},
		AllowHeaders:     []string{"Origin", "Authorization", "Content-Type", "Accept-Encoding"},
		ExposeHeaders:    []string{"Content-Length", "Access-Control-Allow-Origin", "Access-Control-Allow-Credentials", "Access-Control-Allow-Headers", "Access-Control-Allow-Methods"},
		AllowCredentials: true,
	}))
	router.POST("/newchain", GinNewChain)
	router.POST("/addblock", GinAddBlock)
	router.POST("/addtx", GinAddTransaction, timeout.New(
		timeout.WithTimeout(8*time.Second),
		timeout.WithHandler(emptySuccessResponse),
	))
	router.POST("/getblock", GinGetBlocks)
	router.POST("/getlasthash", GinGetLastHash)
	router.POST("/getbalance", GinGetBalance)
	router.POST("/getchainsize", GinGetChainSize)
	router.GET("/getdb", GinGetDb)
	router.POST("/netpush", GinPushBlockToNet)
	router.GET("/check", GinCheck)
	err := router.Run(strings.Trim(ThisServe, "\""))
	if err != nil {
		log.Fatalln(err)
		return
	}
}

func GinNewChain(c *gin.Context) {
	var input *Transport.ChainHelp
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest,
			gin.H{"error": err.Error()})
		return
	} else {
		genesisHash, errChain := NewChain(input.Master, input.Count)
		if errChain != nil {
			responseHash := Transport.CreateHelp{
				Status: errChain.Error(),
			}
			c.JSON(http.StatusBadRequest,
				gin.H{"error": responseHash.Status})
			return
		}
		responseHash := Transport.CreateHelp{
			Status: genesisHash,
		}
		c.JSON(200, gin.H{"GenesisHash": responseHash.Status})
	}
}

func GinAddBlock(c *gin.Context) {
	var input *Transport.BlockHelp
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest,
			gin.H{"error": err.Error()})
		return
	} else {
		block, errAdd := AddBlock(input)
		if errAdd != nil {
			c.JSON(http.StatusBadRequest,
				gin.H{"error": err.Error()})
			return
		}
		c.JSON(200, gin.H{"BlockAddStatus": block})
	}
}

func GinAddTransaction(c *gin.Context) {
	var input *Transport.TransactionHelp
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest,
			gin.H{"error": err.Error()})
		return
	}
	if IsMining {
		c.JSON(200, gin.H{"addTxStatus": "mining"})
	} else {
		transaction, errTx := AddTransaction(input)
		if errTx != nil {
			c.JSON(http.StatusBadRequest,
				gin.H{"error": errTx.Error()})
			return
		}
		c.JSON(200, gin.H{"addTxStatus": transaction})
	}
}

func GinGetBlocks(c *gin.Context) {
	var input *Transport.MasterHelp
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest,
			gin.H{"error": err.Error()})
		return
	} else {
		blocks, errGet := GetBlocks(input)
		if errGet != nil {
			c.JSON(http.StatusBadRequest,
				gin.H{"error": err.Error()})
			return
		}
		c.JSON(200, blocks)
	}
}
func GinGetDb(c *gin.Context) {
	db, err := Blockchain.GetFullDb()
	if err != nil {
		return
	}
	c.JSON(200, db)
}
func GinCheck(c *gin.Context) {
	if IsMining {
		c.JSON(200, gin.H{"addTxStatus": "mining"})
	} else {
		c.JSON(200, gin.H{"addTxStatus": "ready"})
	}

}
func GinGetLastHash(c *gin.Context) {
	var input *Transport.MasterHelp
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest,
			gin.H{"error": err.Error()})
		return
	} else {
		hash, errLH := GetLastHash(input)
		if errLH != nil {
			c.JSON(http.StatusBadRequest,
				gin.H{"error": err.Error()})
			return
		}
		c.JSON(200, gin.H{"LastHash": hash})
	}
}
func GinGetBalance(c *gin.Context) {
	var input *Transport.UserHelp
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest,
			gin.H{"error": err.Error()})
		return
	} else {
		balance, errBalance := GetBalance(input)
		if errBalance != nil {
			c.JSON(http.StatusBadRequest,
				gin.H{"error": err.Error()})
			return
		}
		responseBalance := Transport.BalanceHelp{
			Balance: balance,
		}
		c.JSON(200, gin.H{"Balance": responseBalance.Balance})
	}
}
func GinGetChainSize(c *gin.Context) {
	var input *Transport.MasterHelp
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest,
			gin.H{"error": err.Error()})
		return
	} else {
		size, errChainSize := GetChainSize(input)
		if errChainSize != nil {
			return
		}
		responseSize := Transport.SizeHelp{
			ChainSize: size,
		}
		c.JSON(200, gin.H{"ChainSize": responseSize.ChainSize})
	}
}
func GinPushBlockToNet(c *gin.Context) {
	var input *Transport.BlockHelp
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest,
			gin.H{"error": err.Error()})
		return
	} else {
		errBlock := pushBlockToNet(input)
		if errBlock != nil {
			c.JSON(http.StatusBadRequest,
				gin.H{"error": err.Error()})
			return
		}
		c.JSON(200, gin.H{"Status": "ok"})
	}
}
func ReadAddresses() ([]*fastjson.Value, error) {
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
