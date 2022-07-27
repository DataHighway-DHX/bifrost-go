package base

import "github.com/centrifuge/go-substrate-rpc-client/v4/types"

type HrmpChannelId struct {
	Sender   types.U32
	Receiver types.U32
}
