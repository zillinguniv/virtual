package node

import (
	"net"
	"sync"
	"time"

	"virtual/core"
)

// ----------------------------- Config & Flags -----------------------------

type NodeConfig struct {
	DataDir   string
	GenesisTs int64
	MineEvery time.Duration
	P2PListen string
	PeerSeeds []string
	RPCListen string
	ExtraData string
}

// ----------------------------- Disk Artifacts -----------------------------

type HeadFile struct {
	Hash      string `json:"hash"`
	Timestamp int64  `json:"timestamp"`
	VTCUnix   uint64 `json:"vtcunix"`
	PathUnix  string `json:"pathByUnix"`
	PathHash  string `json:"pathByHash"`
}

type ConfigFile struct {
	GenesisTime int64 `json:"genesisTime"`
}

// ----------------------------- P2P Messaging ------------------------------

type MsgType string

const (
	MsgAnnounce MsgType = "ANNOUNCE"
	MsgGetBlock MsgType = "GETBLOCK"
	MsgBlock    MsgType = "BLOCK"
)

type WireMsg struct {
	Type     MsgType     `json:"type"`
	Hash     string      `json:"hash,omitempty"`
	VTCUnix  uint64      `json:"vtcunix,omitempty"`
	WantHash string      `json:"wantHash,omitempty"`
	Block    *core.Block `json:"block,omitempty"`
}

// ----------------------------- Node State ---------------------------------

type Node struct {
	cfg         NodeConfig
	genesisTime int64

	mu     sync.RWMutex
	head   *core.Block
	headBy HeadFile
	peers  map[string]net.Conn // key = remote addr
}

func New(cfg NodeConfig) *Node {
	return &Node{cfg: cfg, peers: make(map[string]net.Conn)}
}

func (n *Node) GenesisTime() int64 { return n.genesisTime }
