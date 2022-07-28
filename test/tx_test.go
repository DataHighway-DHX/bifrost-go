package test

import (
	"fmt"
	"testing"

	"github.com/JFJun/bifrost-go/client"
	"github.com/JFJun/bifrost-go/crypto"
	"github.com/JFJun/bifrost-go/expand"
	"github.com/JFJun/bifrost-go/tx"
)

func Test_Tx2(t *testing.T) {

	c, err := client.New("")
	if err != nil {
		t.Fatal(err)
	}

	// If the address of some chains (eg: chainX) requires 0xff
	//in front of the byte, then the following value is set to false

	//expand.SetSerDeOptions(false)
	from := ""
	to := ""
	amount := uint64(10000000000)

	// Get the nonce of the from address
	acc, err := c.GetAccountInfo(from)
	if err != nil {
		t.Fatal(err)
	}
	nonce := uint64(acc.Nonce)
	// Create a substrate transaction, this method satisfies all
	// chains that follow the transaction structure of substrate
	transaction := tx.NewSubstrateTransaction(from, nonce)

	// Initialize the metadata expansion structure
	ed, err := expand.NewMetadataExpand(c.Meta)
	if err != nil {
		t.Fatal(err)
	}
	// Initialize the call method of Balances.transfer
	call, err := ed.BalanceTransferCall(to, amount)
	if err != nil {
		t.Fatal(err)
	}
	/*
		//Balances.transfer_keep_alive  call方法
		btkac,err:=ed.BalanceTransferKeepAliveCall(to,amount)
	*/

	/*
		toAmount:=make(map[string]uint64)
		toAmount[to] = amount
		//...
		//true: user Balances.transfer_keep_alive  false: Balances.transfer
		ubtc,err:=ed.UtilityBatchTxCall(toAmount,false)
	*/

	// Set the necessary parameters for the transaction
	transaction.SetGenesisHashAndBlockHash(c.GetGenesisHash(), c.GetGenesisHash()).
		SetSpecAndTxVersion(uint32(c.SpecVersion), uint32(c.TransactionVersion)).
		SetCall(call) //设置call
	// Signed transaction
	sig, err := transaction.SignTransaction("", crypto.Sr25519Type)
	if err != nil {
		t.Fatal(err)
	}

	var result interface{}
	err = c.C.Client.Call(&result, "author_submitExtrinsic", sig)
	if err != nil {
		t.Fatal(err)
	}
	// get txid
	txid := result.(string)
	fmt.Println(txid)
}
