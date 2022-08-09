package test

import (
	"fmt"
	"testing"

	"github.com/DataHighway-DHX/substrate-go/client"
	"github.com/centrifuge/go-substrate-rpc-client/v4/signature"
	"github.com/vedhavyas/go-subkey"
)

func Test_Tx2(t *testing.T) {

	c, err := client.New("wss://tanganika.datahighway.com", true)
	if err != nil {
		t.Fatal(err)
	}

	senderSecret := ""
	recieverAccId := ""
	amount := uint64(10000000000)
	tip := uint64(0)

	fromKp, err := signature.KeyringPairFromSecret(
		senderSecret,
		c.NetId)

	from, err := subkey.SS58Address(fromKp.PublicKey[:], c.NetId)

	fmt.Printf("from : %s to %s amount %d", from, recieverAccId, amount)

	txHash, err := c.AuthorTransferAsset(senderSecret, recieverAccId, amount, tip)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("tx hash %s", txHash.Hex())
}
