package models

type BlockResponse struct {
	Height     int64                `json:"height"`
	ParentHash string               `json:"parent_hash"`
	BlockHash  string               `json:"block_hash"`
	Timestamp  int64                `json:"timestamp"`
	Extrinsic  []*ExtrinsicResponse `json:"extrinsic"`
}

type ExtrinsicResponse struct {
	Type            string `json:"type"`   //Transfer or another
	Status          string `json:"status"` //success or fail
	Txid            string `json:"txid"`
	FromAddress     string `json:"from_address"`
	ToAddress       string `json:"to_address"`
	Amount          string `json:"amount"`
	Fee             string `json:"fee"`
	Signature       string `json:"signature"`
	Nonce           int64  `json:"nonce"`
	Era             string `json:"era"`
	ExtrinsicIndex  int    `json:"extrinsic_index"`
	EventIndex      int    `json:"event_index"`
	ExtrinsicLength int    `json:"extrinsic_length"`
}
