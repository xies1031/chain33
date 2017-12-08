package rpc

type JTransparm struct {
	Execer    string
	Payload   string
	Signature *Signature
	Fee       int64
}

type RawParm struct {
	Data string
}

type Header struct {
	Version    int64
	ParentHash string
	TxHash     string
	StateHash  string
	Height     int64
	BlockTime  int64
}

type Signature struct {
	Ty        int32
	Pubkey    string
	Signature string
}
type Transaction struct {
	Execer    string
	Payload   string
	Signature *Signature
	Fee       int64
	Expire    int32
	Nonce     int32
	To        string
}
type ReceiptLog struct {
	Ty  int32
	Log string
}
type ReceiptData struct {
	Ty   int32
	Logs []*ReceiptLog
}
type Block struct {
	Version    int64
	ParentHash string
	TxHash     string
	StateHash  string
	Height     int64
	BlockTime  int64
	Txs        []*Transaction
}
type BlockDetail struct {
	Block    *Block
	Receipts []*ReceiptData
}

type BlockDetails struct {
	Items []*BlockDetail
}

type TransactionDetail struct {
	Tx      *Transaction
	Receipt *ReceiptData
	Proofs  []string
}

type ReplyTxInfo struct {
	Hash   string
	Height int64
	Index  int64
}
type TransactionDetails struct {
	Txs []*Transaction
}

type ReplyTxList struct {
	Txs []*Transaction
}
type ReplyHash struct {
	Hash string
}
type ReplyHashes struct {
	Hashes []string
}
type PeerList struct {
	Peers []*Peer
}
type Peer struct {
	Addr        string
	Port        int32
	Name        string
	MempoolSize int32
	Header      *Header
}
