package client

import (
	"fmt"

	"github.com/JFJun/go-substrate-crypto/ss58"

	"strings"

	gsrc "github.com/centrifuge/go-substrate-rpc-client/v4"
	gsClient "github.com/centrifuge/go-substrate-rpc-client/v4/client"
	"github.com/centrifuge/go-substrate-rpc-client/v4/rpc"
	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
)

type Client struct {
	C                  *gsrc.SubstrateAPI
	Meta               *types.Metadata
	prefix             []byte
	ChainName          string
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

	if specVersion != c.SpecVersion {
		c.Meta, err = c.C.RPC.State.GetMetadataLatest()
		if err != nil {
			return fmt.Errorf("init metadata error: %v", err)
		}
		c.SpecVersion = specVersion
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
	ci.Chain, err = c.C.RPC.System.Chain()
	if err != nil {
		return nil, fmt.Errorf("cannot get chain info: %v", err)
	}
	ci.NodeName, err = c.C.RPC.System.Name()
	if err != nil {
		return nil, fmt.Errorf("cannot get name info: %v", err)
	}
	ci.NodeVersion, err = c.C.RPC.System.Version()
	if err != nil {
		return nil, fmt.Errorf("cannot get version info: %v", err)
	}
	return
}

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

// Customize the prefix. If the prefix loaded at startup is wrong, you need to configure the prefix manually
func (c *Client) SetPrefix(prefix []byte) {
	c.prefix = prefix
}
