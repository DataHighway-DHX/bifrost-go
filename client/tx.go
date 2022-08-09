package client

import (
	"errors"
	"fmt"

	"github.com/centrifuge/go-substrate-rpc-client/v4/signature"
	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
	"github.com/decred/base58"
)

func (c *Client) AuthorTransferAsset(senderSecret, recieverAccId string, value, tip uint64) (txHash types.Hash, err error) {
	from, err := signature.KeyringPairFromSecret(
		senderSecret,
		c.NetId)
	if err != nil {
		return txHash, fmt.Errorf("can't get sender key pair %v", err)
	}

	to, err := types.NewMultiAddressFromHexAccountID(recieverAccId)
	if err != nil {
		return txHash, fmt.Errorf("can't get reciever multi address %v", err)
	}

	amount := types.NewUCompactFromUInt(value)
	c.Meta, err = c.API.RPC.State.GetMetadataLatest()
	if err != nil {
		return txHash, fmt.Errorf("can't get latest metadata %v", err)
	}

	so, err := c.GetSignatureOptions(from, tip)
	if err != nil {
		return txHash, fmt.Errorf("can't get signature options %v", err)
	}

	ca, err := NewCall(c.Meta, "Balances.transfer", to, amount)
	if err != nil {
		return txHash, fmt.Errorf("can't get Balances.transfer call from metadata %v", err)
	}

	ext := types.NewExtrinsic(ca)

	err = ext.Sign(from, so)
	if err != nil {
		return txHash, fmt.Errorf("can't sign extrinsic %v", err)
	}

	txHash, err = c.API.RPC.Author.SubmitExtrinsic(ext)
	if err != nil {
		return txHash, fmt.Errorf("can't SubmitExtrinsic %v", err)
	}
	return
}

func (c *Client) GetSignatureOptions(signer signature.KeyringPair, tip uint64) (so types.SignatureOptions, err error) {
	gHash, err := c.GetGenesisHash()
	if err != nil {
		return so, err
	}
	ai, err := c.GetAccountInfo(signer)
	if err != nil {
		return so, err
	}
	err = c.checkRuntimeVersion()
	if err != nil {
		return so, err
	}
	rv := c.RuntimeVersion
	so = types.SignatureOptions{
		BlockHash:          *gHash,
		Era:                types.ExtrinsicEra{IsMortalEra: false, IsImmortalEra: true},
		GenesisHash:        *gHash,
		Nonce:              types.NewUCompactFromUInt(uint64(ai.Nonce)),
		SpecVersion:        rv.SpecVersion,
		Tip:                types.NewUCompactFromUInt(tip),
		TransactionVersion: rv.TransactionVersion,
	}
	return
}

func NewCall(m *types.Metadata, call string, args ...interface{}) (types.Call, error) {
	c, err := m.FindCallIndex(call)
	if err != nil {
		return types.Call{}, err
	}
	c = types.CallIndex{
		SectionIndex: c.MethodIndex,
		MethodIndex:  c.SectionIndex,
	}

	var a []byte
	ci, err := types.Encode(c.SectionIndex)
	if err != nil {
		return types.Call{}, err
	}
	a = append(a, ci...)

	for _, arg := range args {
		e, err := types.Encode(arg)
		if err != nil {
			return types.Call{}, err
		}
		a = append(a, e...)
	}

	return types.Call{
		CallIndex: c,
		Args:      a,
	}, nil
}

func DecodeToPub(address string) ([]byte, error) {
	data := base58.Decode(address)
	if len(data) != 35 {
		return nil, errors.New("base58 decode error")
	}
	return data[1 : len(data)-2], nil
}
