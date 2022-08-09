package test

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/DataHighway-DHX/substrate-go/client"
)

func Test_GetBlockByNumber(t *testing.T) {

	c, err := client.New("wss://tanganika.datahighway.com", true)
	if err != nil {
		t.Fatal(err)
	}
	// c.SetPrefix(ss58.DataHighwayPrefix)

	resp, err := c.GetBlockByNumber(13953)
	if err != nil {
		t.Fatal(err)
	}

	d, _ := json.Marshal(resp)
	fmt.Println(string(d))

}
