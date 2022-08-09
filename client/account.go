package client

import (
	"fmt"

	"github.com/centrifuge/go-substrate-rpc-client/v4/signature"
	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
)

func (c *Client) GetAccountInfo(acc signature.KeyringPair) (*types.AccountInfo, error) {
	key, err := types.CreateStorageKey(c.Meta, "System", "Account", acc.PublicKey, nil)
	if err != nil {
		return nil, fmt.Errorf("can't create storage key %v", err)
	}
	var accountInfo types.AccountInfo
	ok, err := c.API.RPC.State.GetStorageLatest(key, &accountInfo)
	if err != nil || !ok {
		return nil, fmt.Errorf("can't get latest storage for account %v  ok:%v", err, ok)
	}
	return &accountInfo, nil
}
