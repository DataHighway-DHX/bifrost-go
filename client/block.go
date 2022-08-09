package client

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"time"

	"github.com/DataHighway-DHX/substrate-go/models"
	"github.com/centrifuge/go-substrate-rpc-client/v4/scale"
	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
	"golang.org/x/crypto/blake2b"
)

/*
根据height解析block，返回block是否包含交易
*/
func (c *Client) GetBlockByNumber(height int64) (*models.BlockResponse, error) {
	hash, err := c.API.RPC.Chain.GetBlockHash(uint64(height))
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
	block, err = c.API.RPC.Chain.GetBlock(blockHash)
	if err != nil {
		return nil, fmt.Errorf("get block error: %v", err)
	}
	blockResp := new(models.BlockResponse)

	number := int64(block.Block.Header.Number)
	blockResp.Height = number
	blockResp.ParentHash = block.Block.Header.ParentHash.Hex()
	blockResp.BlockHash = blockHash.Hex()

	ts, err := c.getBlockTimestamp(block.Block.Extrinsics)
	if err != nil {
		return nil, fmt.Errorf("unable to get block timestamp: %v", err)
	}
	blockResp.Timestamp = ts.Unix()

	blockResp.Extrinsic, err = c.parseExtrinsic(blockHash, block.Block.Header.ParentHash, block.Block.Extrinsics)
	if err != nil {
		return nil, err
	}

	return blockResp, nil
}

func (c *Client) parseExtrinsic(blockHash, parentHash types.Hash, extrinsics []types.Extrinsic) ([]*models.ExtrinsicResponse, error) {
	var (
		eventKey types.StorageKey
		err      error
	)
	exts := []*models.ExtrinsicResponse{}
	if len(extrinsics) == 0 {
		return exts, nil
	}

	c.Meta, err = c.API.RPC.State.GetMetadataLatest()
	if err != nil {
		return nil, fmt.Errorf("cannot fetch latest metadata %v", err)
	}

	eventKey, err = types.CreateStorageKey(c.Meta, "System", "Events")
	if err != nil {
		return nil, fmt.Errorf("unable to create storage key:%v", err)
	}

	raw, err := c.API.RPC.State.GetStorageRaw(eventKey, blockHash)
	if err != nil {
		return nil, fmt.Errorf("unable to query storage: %v", err)
	}

	var events types.EventRecords
	err = (*types.EventRecordsRaw)(raw).DecodeEventRecords(c.Meta, &events)
	if err != nil {
		return nil, fmt.Errorf("unable to decode event records: %v", err)
	}

	for _, tr := range events.Balances_Transfer {
		if !(len(extrinsics) > int(tr.Phase.AsApplyExtrinsic)) {
			return nil, fmt.Errorf("unable to access extrinsics by index: %d", tr.Phase.AsApplyExtrinsic)
		}
		currentExt := extrinsics[tr.Phase.AsApplyExtrinsic]
		fee, err := c.getPartialFee(currentExt, parentHash.Hex())
		if err != nil {
			return nil, fmt.Errorf("unable to get block timestamp: %v", err)
		}

		td, err := txDataFromExtrinsic(currentExt)
		if err != nil {
			return nil, fmt.Errorf("error while getting tx info from extrinsic: %v", err)
		}

		exts = append(exts,
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

	return exts, nil
}

type txData struct {
	txid, era, sig string
	len            int
}

func txDataFromExtrinsic(ext types.Extrinsic) (td *txData, err error) {
	td = &txData{}
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

func (c *Client) getPartialFee(ext types.Extrinsic, parentHash string) (string, error) {
	var result map[string]interface{}
	err := c.API.Client.Call(&result, "payment_queryInfo", ext, parentHash)
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

func (c *Client) getBlockTimestamp(ext []types.Extrinsic) (*time.Time, error) {

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
