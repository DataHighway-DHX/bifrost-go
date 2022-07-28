package client

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"time"

	"github.com/JFJun/bifrost-go/models"
	"github.com/JFJun/go-substrate-crypto/ss58"

	"strings"

	gsrc "github.com/centrifuge/go-substrate-rpc-client/v4"
	gsClient "github.com/centrifuge/go-substrate-rpc-client/v4/client"
	"github.com/centrifuge/go-substrate-rpc-client/v4/rpc"
	"github.com/centrifuge/go-substrate-rpc-client/v4/scale"
	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
	"golang.org/x/crypto/blake2b"
)

type Client struct {
	C                  *gsrc.SubstrateAPI
	Meta               *types.Metadata
	prefix             []byte //币种的前缀
	ChainName          string //链名字
	SpecVersion        int
	TransactionVersion int
	genesisHash        string
	url                string
}

func New(url string) (*Client, error) {
	c := new(Client)
	c.url = url
	var err error

	// 初始化rpc客户端
	c.C, err = gsrc.NewSubstrateAPI(url)
	if err != nil {
		return nil, err
	}
	//检查当前链运行的版本
	err = c.checkRuntimeVersion()
	if err != nil {
		return nil, err
	}
	c.prefix = ss58.BifrostPrefix
	return c, nil
}

func (c *Client) reConnectWs() (*gsrc.SubstrateAPI, error) {
	cl, err := gsClient.Connect(c.url)
	if err != nil {
		return nil, err
	}
	newRPC, err := rpc.NewRPC(cl)
	if err != nil {
		return nil, err
	}
	return &gsrc.SubstrateAPI{
		RPC:    newRPC,
		Client: cl,
	}, nil
}

func (c *Client) checkRuntimeVersion() error {
	v, err := c.C.RPC.State.GetRuntimeVersionLatest()
	if err != nil {
		if !strings.Contains(err.Error(), "tls: use of closed connection") {
			return fmt.Errorf("init runtime version error,err=%v", err)
		}
		//	重连处理，这是因为第三方包的问题，所以只能这样处理了了
		cl, err := c.reConnectWs()
		if err != nil {
			return fmt.Errorf("reconnect error: %v", err)
		}
		c.C = cl
		v, err = c.C.RPC.State.GetRuntimeVersionLatest()
		if err != nil {
			return fmt.Errorf("init runtime version error,aleady reconnect,err: %v", err)
		}
	}
	c.TransactionVersion = int(v.TransactionVersion)
	c.ChainName = v.SpecName
	specVersion := int(v.SpecVersion)
	//检查metadata数据是否有升级
	if specVersion != c.SpecVersion {
		c.Meta, err = c.C.RPC.State.GetMetadataLatest()
		if err != nil {
			return fmt.Errorf("init metadata error: %v", err)
		}
		c.SpecVersion = specVersion
	}
	return nil
}

/*
获取创世区块hash
*/
func (c *Client) GetGenesisHash() string {
	if c.genesisHash != "" {
		return c.genesisHash
	}
	hash, err := c.C.RPC.Chain.GetBlockHash(0)
	if err != nil {
		return ""
	}
	c.genesisHash = hash.Hex()
	return hash.Hex()
}

/*
自定义设置prefix，如果启动时加载的prefix是错误的，则需要手动配置prefix
*/
func (c *Client) SetPrefix(prefix []byte) {
	c.prefix = prefix
}

/*
根据height解析block，返回block是否包含交易
*/
func (c *Client) GetBlockByNumber(height int64) (*models.BlockResponse, error) {
	hash, err := c.C.RPC.Chain.GetBlockHash(uint64(height))
	if err != nil {
		return nil, fmt.Errorf("get block hash error:%v,height:%d", err, height)
	}
	return c.GetBlockByHash(hash)
}

/*
根据blockHash解析block，返回block是否包含交易
*/
func (c *Client) GetBlockByHash(blockHash types.Hash) (*models.BlockResponse, error) {
	var (
		block *types.SignedBlock
		err   error
	)
	err = c.checkRuntimeVersion()
	if err != nil {
		return nil, err
	}
	block, err = c.C.RPC.Chain.GetBlock(blockHash)
	if err != nil {
		return nil, fmt.Errorf("get block error: %v", err)
	}
	blockResp := new(models.BlockResponse)

	number := int64(block.Block.Header.Number)
	blockResp.Height = number
	blockResp.ParentHash = block.Block.Header.ParentHash.Hex()
	blockResp.BlockHash = blockHash.Hex()

	if len(block.Block.Extrinsics) > 0 {
		err = c.parseExtrinsicByStorage(blockHash, blockResp, block.Block.Extrinsics)
		if err != nil {
			return nil, err
		}
	}
	return blockResp, nil
}

type parseBlockExtrinsicParams struct {
	from, to, sig, era, txid, fee string
	nonce                         int64
	extrinsicIdx, length          int
}

/*
解析外部交易extrinsic
*/
func (c *Client) parseExtrinsicByStorage(blockHash types.Hash, blockResp *models.BlockResponse, extrinsics []types.Extrinsic) error {
	var (
		eventKey types.StorageKey
		err      error
	)

	c.Meta, err = c.C.RPC.State.GetMetadataLatest()
	if err != nil {
		return fmt.Errorf("cannot fetch latest metadata %v", err)
	}

	eventKey, err = types.CreateStorageKey(c.Meta, "System", "Events")
	if err != nil {
		return fmt.Errorf("unable to create storage key:%v", err)
	}

	raw, err := c.C.RPC.State.GetStorageRaw(eventKey, blockHash)
	if err != nil {
		return fmt.Errorf("unable to query storage: %v", err)
	}

	var events types.EventRecords
	err = (*types.EventRecordsRaw)(raw).DecodeEventRecords(c.Meta, &events)
	if err != nil {
		return fmt.Errorf("unable to decode event records: %v", err)
	}

	ts, err := c.GetBlockTimestamp(extrinsics)
	if err != nil {
		return fmt.Errorf("unable to get block timestamp: %v", err)
	}
	blockResp.Timestamp = ts.Unix()

	for _, tr := range events.Balances_Transfer {
		if !(len(extrinsics) > int(tr.Phase.AsApplyExtrinsic)) {
			return fmt.Errorf("unable to access extrinsics by index: %d", tr.Phase.AsApplyExtrinsic)
		}
		currentExt := extrinsics[tr.Phase.AsApplyExtrinsic]
		fee, err := c.getPartialFee(currentExt, blockResp.ParentHash)
		if err != nil {
			return fmt.Errorf("unable to get block timestamp: %v", err)
		}

		td, err := txDataFromExtrinsic(currentExt)
		if err != nil {
			return fmt.Errorf("error while getting tx info from extrinsic: %v", err)
		}

		blockResp.Extrinsic = append(blockResp.Extrinsic,
			&models.ExtrinsicResponse{
				Type:            "transfer",
				Status:          "success",
				Amount:          tr.Value.String(),
				FromAddress:     fmt.Sprintf("%#x", tr.From),
				ToAddress:       fmt.Sprintf("%#x", tr.To),
				EventIndex:      int(tr.Phase.AsApplyExtrinsic),
				Txid:            td.txid,
				Fee:             fee,
				Era:             td.era,
				Signature:       td.sig,
				Nonce:           currentExt.Signature.Nonce.Int64(),
				ExtrinsicIndex:  int(tr.Phase.AsApplyExtrinsic),
				ExtrinsicLength: td.len,
			})
	}

	return nil
}

type txData struct {
	txid, era, sig string
	len            int
}

func txDataFromExtrinsic(ext types.Extrinsic) (td *txData, err error) {
	td.txid, err = getTxId(ext)
	if err != nil {
		return nil, fmt.Errorf("unable to get txid: %v", err)
	}
	td.era, err = getEra(ext)
	if err != nil {
		return nil, fmt.Errorf("unable to get era: %v", err)
	}
	td.sig, err = getSignature(ext)
	if err != nil {
		return nil, fmt.Errorf("unable to get signature: %v", err)
	}
	td.len, err = getLength(ext)
	if err != nil {
		return nil, fmt.Errorf("unable to get extrinsic length: %v", err)
	}
	return
}

func getTxId(ext types.Extrinsic) (string, error) {
	extBytes, err := types.Encode(ext)
	if err != nil {
		return "", fmt.Errorf("failed to encode extrinsic")
	}
	d := blake2b.Sum256(extBytes)

	return "0x" + hex.EncodeToString(d[:]), nil
}

func getEra(ext types.Extrinsic) (string, error) {
	era, err := types.Encode(ext.Signature.Era)
	if err != nil {
		return "", fmt.Errorf("failed to encode signature era")
	}
	return fmt.Sprintf("%#x", era), nil
}

func getSignature(ext types.Extrinsic) (string, error) {
	if ext.Signature.Signature.IsEcdsa {
		return ext.Signature.Signature.AsEcdsa.Hex(), nil
	}
	if ext.Signature.Signature.IsEd25519 {
		return ext.Signature.Signature.AsEd25519.Hex(), nil
	}
	if ext.Signature.Signature.IsSr25519 {
		return ext.Signature.Signature.AsSr25519.Hex(), nil
	}
	return "", fmt.Errorf("can't get signature")
}

func getLength(ext types.Extrinsic) (int, error) {
	var bb = bytes.Buffer{}
	tempEnc := scale.NewEncoder(&bb)

	// encode the version of the extrinsic
	err := tempEnc.Encode(ext.Version)
	if err != nil {
		return 0, fmt.Errorf("failed to encode Version")
	}
	// encode the signature if signed
	if ext.IsSigned() {
		err = tempEnc.Encode(ext.Signature)
		if err != nil {
			return 0, fmt.Errorf("failed to encode Signature")
		}

	}
	// encode the method
	err = tempEnc.Encode(ext.Method)
	if err != nil {
		return 0, fmt.Errorf("failed to encode Method")
	}

	// take the temporary buffer to determine length, write that as prefix
	eb := bb.Bytes()

	return len(eb), nil
}

/*
获取外部交易extrinsic的手续费
*/
func (c *Client) getPartialFee(ext types.Extrinsic, parentHash string) (string, error) {
	var result map[string]interface{}
	err := c.C.Client.Call(&result, "payment_queryInfo", ext, parentHash)
	if err != nil {
		return "", fmt.Errorf("get payment info error: %v", err)
	}
	if result["partialFee"] == nil {
		return "", errors.New("result partialFee is nil ptr")
	}
	fee, ok := result["partialFee"].(string)
	if !ok {
		return "", fmt.Errorf("partialFee is not string type: %v", result["partialFee"])
	}
	return fee, nil
}

func (c *Client) GetBlockTimestamp(ext []types.Extrinsic) (*time.Time, error) {

	// callIndex needed to find correct Extrinsic
	callIndex, err := c.Meta.FindCallIndex("Timestamp.set")
	if err != nil {
		return nil, err
	}

	timestamp := new(big.Int)
	for _, extrinsic := range ext {
		if extrinsic.Method.CallIndex != callIndex {
			continue
		}
		timeDecoder := scale.NewDecoder(bytes.NewReader(extrinsic.Method.Args))
		timestamp, err = timeDecoder.DecodeUintCompact()
		if err != nil {
			return nil, err
		}
		break
	}
	msec := timestamp.Int64()
	time := time.Unix(msec/1e3, (msec%1e3)*1e6)
	return &time, nil
}
