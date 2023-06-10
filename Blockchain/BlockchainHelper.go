package Blockchain

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	//"encoding/json"
	"github.com/goccy/go-json"
	"github.com/golang/protobuf/ptypes/timestamp"
	"github.com/google/uuid"
	"github.com/spf13/viper"
	"github.com/valyala/fastjson"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"io"
	"log"
	"math"
	"math/big"
	mr "math/rand"
	"net/http"
	"strconv"
	"time"
)

// SerializeBlock TODO Rewrite
func SerializeBlock(block *Block) (string, error) {
	jsonData, err := json.MarshalIndent(*block, "", "\t")
	if err != nil {
		return "", err
	}
	return string(jsonData), nil
}

// DeserializeBlock TODO Rewrite
func DeserializeBlock(data string) (*Block, error) {
	var block Block
	err := json.Unmarshal([]byte(data), &block)
	if err != nil {
		return nil, err
	}
	return &block, nil
}

func GetTime() (*timestamp.Timestamp, error) {
	TimeUrl := viper.GetString("TIME_URL")
	resp, err := http.Get(TimeUrl)
	if err != nil {
		defer func(Body io.ReadCloser) {
			err := Body.Close()
			if err != nil {
				log.Fatal(err)
			}
		}(resp.Body)
		return timestamppb.New(time.Time{}), err
	}
	//TODO what is log fatal
	defer func(Body io.ReadCloser) {
		errReadCloser := Body.Close()
		if errReadCloser != nil {
			log.Fatal(errReadCloser)
		}
	}(resp.Body)
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return timestamppb.New(time.Time{}), err
	}
	var p fastjson.Parser
	v, err := p.Parse(string(body))
	if err != nil {
		return timestamppb.New(time.Time{}), err
	}
	parsedTime, err := time.Parse("2006-01-02T15:04:05", string(v.GetStringBytes("dateTime")))
	if err != nil {
		return timestamppb.New(time.Time{}), err
	}
	return timestamppb.New(parsedTime), nil

}

//func SetHash(data string) string {
//	hash := sha256.Sum256([]byte(data))
//	return string(hash[:])
//}

func ImportToDB(PrivateKey string, PublicKey string) error {
	db, err := gorm.Open(sqlite.Open("Database/ContractDB.db"), &gorm.Config{})
	if err != nil {
		return err
	}
	db.Exec("INSERT INTO KeyLinks (Id, PublicKey, PrivateKey) VALUES ($1, $2, $3)",
		uuid.NewString(),
		PublicKey,
		PrivateKey)
	return nil
}

func Sign(privateKey string, data string) string {
	//tempSign := bytes.Join([][]byte{
	//	[]byte(privateKey),
	//	[]byte(data),
	//},
	//	[]byte{})
	tempSign := privateKey + data
	signature := HashSum(tempSign)
	return signature
}

func ToBytes(data int64) []byte {
	var buf = new(bytes.Buffer)
	err := binary.Write(buf, binary.BigEndian, data)
	if err != nil {
		return nil
	}
	return buf.Bytes()
}

func GenerateRandomBytes(max uint64) ([]byte, error) {
	var slice = make([]byte, max)
	_, err := rand.Read(slice)
	if err != nil {
		return nil, err
	}
	return slice, nil
}

func HashSum(data string) string {
	hash := sha256.Sum256([]byte(data))
	return fmt.Sprintf("%x", hash[:])
}

func ProofOfWork(blockHash string, difficulty uint8, ch chan bool) uint64 {
	RandomGenerator := mr.New(mr.NewSource(time.Now().Unix()))
	var Target = big.NewInt(1)
	var intHash = big.NewInt(1)
	var nonce = uint64(RandomGenerator.Uint32())
	var hash string
	Target.Lsh(Target, 256-uint(difficulty))
	for nonce < math.MaxUint64 {
		select {
		case <-ch:
			if true {
				fmt.Println()
			}
			return nonce
		default:
			hash = HashSum(strconv.FormatUint(nonce, 10) + blockHash)
			if true {
				fmt.Printf("\rMining: %v", hash)
			}
			decodeString, err := hex.DecodeString(hash)
			if err != nil {
				return 0
			}
			intHash.SetBytes(decodeString)
			if intHash.Cmp(Target) == -1 {
				if true {
					fmt.Println()
				}
				return nonce
			}
			nonce++
		}
	}
	return nonce
}
