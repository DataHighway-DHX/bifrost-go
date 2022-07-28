package client

import (
	"fmt"
	"strings"

	"github.com/JFJun/bifrost-go/expand"
	"github.com/JFJun/go-substrate-crypto/ss58"
	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
)

func (c *Client) GetAccountInfo(address string) (*types.AccountInfo, error) {
	var (
		storage types.StorageKey
		err     error
		pub     []byte
	)
	defer func() {
		if err1 := recover(); err1 != nil {
			err = fmt.Errorf("panic decode event: %v", err1)
		}
	}()
	err = c.checkRuntimeVersion()
	if err != nil {
		return nil, err
	}
	pub, err = ss58.DecodeToPub(address)
	if err != nil {
		return nil, fmt.Errorf("ss58 decode address error: %v", err)
	}
	storage, err = types.CreateStorageKey(c.Meta, "System", "Account", pub, nil)
	if err != nil {
		return nil, fmt.Errorf("create System.Account storage error: %v", err)
	}
	var accountInfo types.AccountInfo
	var ok bool
	switch strings.ToLower(c.ChainName) {

	case "polkadot", "kusama":
		var accountInfoProviders expand.AccountInfoWithProviders
		ok, err = c.C.RPC.State.GetStorageLatest(storage, &accountInfoProviders)
		if err != nil || !ok {
			return nil, fmt.Errorf("get account info error: %v", err)
		}
		accountInfo.Nonce = accountInfoProviders.Nonce
		accountInfo.Data.Free = accountInfoProviders.Data.Free
		accountInfo.Data.FreeFrozen = accountInfoProviders.Data.FreeFrozen
		accountInfo.Data.MiscFrozen = accountInfoProviders.Data.MiscFrozen
		accountInfo.Data.Reserved = accountInfoProviders.Data.Reserved
	default:
		ok, err = c.C.RPC.State.GetStorageLatest(storage, &accountInfo)
		if err != nil || !ok {
			return nil, fmt.Errorf("get account info error: %v", err)
		}
	}

	return &accountInfo, nil
}
