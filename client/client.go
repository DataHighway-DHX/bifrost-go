package client

import (
	"fmt"

	"github.com/DataHighway-DHX/substrate-go/base"

	"strings"

	gsrc "github.com/centrifuge/go-substrate-rpc-client/v4"
	gsClient "github.com/centrifuge/go-substrate-rpc-client/v4/client"
	"github.com/centrifuge/go-substrate-rpc-client/v4/rpc"
	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
)

type Client struct {
	API            *gsrc.SubstrateAPI
	Meta           *types.Metadata
	BasicType      *base.BasicTypes
	RuntimeVersion *types.RuntimeVersion
	prefix         []byte
	genesisHash    types.Hash
	url            string
	NetId          uint8
}

func New(url string, noPalletIndices bool) (*Client, error) {
	c := new(Client)
	c.url = url
	var err error

	c.BasicType, err = base.InitBasicTypesByHexData()
	if err != nil {
		return nil, fmt.Errorf("init base type error: %v", err)
	}

	c.API, err = gsrc.NewSubstrateAPI(url)
	if err != nil {
		return nil, err
	}

	err = c.checkRuntimeVersion()
	if err != nil {
		return nil, err
	}

	netId, err := c.BasicType.GetNetworkId(c.RuntimeVersion.SpecName)
	if err != nil {
		return nil, err
	}
	c.NetId = netId
	// expand.SetSerDeOptions(noPalletIndices)
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
	v, err := c.API.RPC.State.GetRuntimeVersionLatest()
	if err != nil {
		if !strings.Contains(err.Error(), "tls: use of closed connection") {
			return fmt.Errorf("init runtime version error,err=%v", err)
		}
		cl, err := c.reConnectWs()
		if err != nil {
			return fmt.Errorf("reconnect error: %v", err)
		}
		c.API = cl
		v, err = c.API.RPC.State.GetRuntimeVersionLatest()
		if err != nil {
			return fmt.Errorf("init runtime version error, already reconnect,err: %v", err)
		}
	}

	if c.RuntimeVersion != v {
		c.Meta, err = c.API.RPC.State.GetMetadataLatest()
		if err != nil {
			return fmt.Errorf("init metadata error: %v", err)
		}
		c.RuntimeVersion = v
	}
	return nil
}

type ChainInfo struct {
	Chain       types.Text
	NodeName    types.Text
	NodeVersion types.Text
}

func (c *Client) ChainInfo() (ci *ChainInfo, err error) {
	ci = &ChainInfo{}
	ci.Chain, err = c.API.RPC.System.Chain()
	if err != nil {
		return nil, fmt.Errorf("cannot get chain info: %v", err)
	}
	ci.NodeName, err = c.API.RPC.System.Name()
	if err != nil {
		return nil, fmt.Errorf("cannot get name info: %v", err)
	}
	ci.NodeVersion, err = c.API.RPC.System.Version()
	if err != nil {
		return nil, fmt.Errorf("cannot get version info: %v", err)
	}
	return
}

func (c *Client) GetGenesisHash() (*types.Hash, error) {
	emptyH := types.Hash{}
	if c.genesisHash.Hex() != emptyH.Hex() {
		return &c.genesisHash, nil
	}
	hash, err := c.API.RPC.Chain.GetBlockHash(0)
	if err != nil {
		return nil, fmt.Errorf("can't get genesis hash")
	}
	c.genesisHash = hash
	return &c.genesisHash, nil
}

// Customize the prefix. If the prefix loaded at startup is wrong, you need to configure the prefix manually
func (c *Client) SetPrefix(prefix []byte) {
	c.prefix = prefix
}
