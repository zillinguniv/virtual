package core

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

// ---------------------------- Types & Constants ----------------------------

type Hash = string

const ZeroHash Hash = "0x0000000000000000000000000000000000000000000000000000000000000000"

// BlockHeader is a fuller VTC header inspired by Ethereum, adapted for a timechain.
type BlockHeader struct {
	ParentHash       Hash   `json:"parentHash"`
	Timestamp        int64  `json:"timestamp"`
	VTCUnix          uint64 `json:"vtcunix"`
	StateRoot        Hash   `json:"stateRoot"`
	TransactionsRoot Hash   `json:"transactionsRoot"`
	ReceiptsRoot     Hash   `json:"receiptsRoot"`
	EventsRoot       Hash   `json:"eventsRoot"`
	ExtraData        string `json:"extraData,omitempty"`
}

// Block wraps the header and its hash (hash(header)).
type Block struct {
	Header BlockHeader `json:"header"`
	Hash   Hash        `json:"hash"`
}

// Roots groups input roots when building a new block.
type Roots struct {
	StateRoot        Hash
	TransactionsRoot Hash
	ReceiptsRoot     Hash
	EventsRoot       Hash
}

// ------------------------------- Hashing ----------------------------------

func hashHeader(h BlockHeader) Hash {
	b, _ := json.Marshal(h)
	sum := sha256.Sum256(b)
	return "0x" + hex.EncodeToString(sum[:])
}

// ------------------------------ Construction ------------------------------

// NewGenesisFull creates the genesis block with zero roots (or provided via Roots).
func NewGenesisFull(genesisTime int64, extra string, roots Roots, literal bool) *Block {
	h := BlockHeader{
		ParentHash:       ZeroHash,
		Timestamp:        genesisTime,
		VTCUnix:          0,
		StateRoot:        orZero(roots.StateRoot),
		TransactionsRoot: orZero(roots.TransactionsRoot),
		ReceiptsRoot:     orZero(roots.ReceiptsRoot),
		EventsRoot:       orZero(roots.EventsRoot),
		ExtraData:        extra,
	}
	b := &Block{Header: h}
	if literal {
		b.Hash = "0xGENESIS"
	} else {
		b.Hash = hashHeader(h)
	}
	return b
}

// NewBlockFull builds a child block.
func NewBlockFull(parent *Block, ts int64, genesisTime int64, roots Roots, extra string) (*Block, error) {
	if parent == nil {
		return nil, errors.New("parent is nil")
	}
	if ts < parent.Header.Timestamp {
		return nil, errors.New("timestamp must be >= parent timestamp")
	}
	var u uint64
	if ts > genesisTime {
		u = uint64(ts - genesisTime)
	}
	h := BlockHeader{
		ParentHash:       parent.Hash,
		Timestamp:        ts,
		VTCUnix:          u,
		StateRoot:        orZero(roots.StateRoot),
		TransactionsRoot: orZero(roots.TransactionsRoot),
		ReceiptsRoot:     orZero(roots.ReceiptsRoot),
		EventsRoot:       orZero(roots.EventsRoot),
		ExtraData:        extra,
	}
	return &Block{Header: h, Hash: hashHeader(h)}, nil
}

func orZero(h Hash) Hash {
	if h == "" {
		return ZeroHash
	}
	return h
}

// ------------------------------- Persistence (simple) ----------------------

// Layout:
// storage/block/
//   by-unix/<vtcunix>.json   // berisi *hash saja* (JSON string)
//   by-hash/<hash>.json      // contains full block (header + hash)

const (
	BlocksRootSimple = "storage/block"
	DirByUnix        = "by-unix"
	DirByHash        = "by-hash"
)

func ensureDirSimple(dir string) error { return os.MkdirAll(dir, 0o755) }

func pathByUnix(u uint64) (string, error) {
	dir := filepath.Join(BlocksRootSimple, DirByUnix)
	if err := ensureDirSimple(dir); err != nil {
		return "", err
	}
	fname := strconv.FormatUint(u, 10) + ".json"
	return filepath.Join(dir, fname), nil
}

func pathByHash(h Hash) (string, error) {
	dir := filepath.Join(BlocksRootSimple, DirByHash)
	if err := ensureDirSimple(dir); err != nil {
		return "", err
	}
	fname := sanitizeHash(h) + ".json"
	return filepath.Join(dir, fname), nil
}

// helper: tulis JSON terformat
func writeJSON(path string, v any) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

// SaveBlockSimple:
// - by-hash: simpan block lengkap (JSON object)
// - by-unix: simpan *hash saja* (JSON string)
func SaveBlockSimple(b *Block) (unixPath string, hashPath string, err error) {
	up, err := pathByUnix(b.Header.VTCUnix)
	if err != nil {
		return "", "", err
	}
	hp, err := pathByHash(b.Hash)
	if err != nil {
		return "", "", err
	}

	// by-hash -> full block
	if err := writeJSON(hp, b); err != nil {
		return "", "", err
	}
	// by-unix -> hash only (JSON string)
	if err := writeJSON(up, b.Hash); err != nil {
		return "", "", err
	}
	return up, hp, nil
}

// Convenience wrapper: return path by-hash
func SaveBlock(b *Block) (string, error) {
	_, hp, err := SaveBlockSimple(b)
	return hp, err
}

// Loaders -------------------------------------------------------------------

// LoadBlockByUnix: baca hash (JSON string) → lanjut load lengkap via by-hash
func LoadBlockByUnix(u uint64) (*Block, error) {
	p, err := pathByUnix(u)
	if err != nil {
		return nil, err
	}
	raw, err := os.ReadFile(p)
	if err != nil {
		return nil, err
	}
	var h Hash
	if err := json.Unmarshal(raw, &h); err != nil {
		return nil, fmt.Errorf("failed to parse hash at %s: %w", p, err)
	}
	return LoadBlockByHash(h)
}

// LoadBlockByHash: baca block lengkap dari by-hash
func LoadBlockByHash(h Hash) (*Block, error) {
	p, err := pathByHash(h)
	if err != nil {
		return nil, err
	}
	raw, err := os.ReadFile(p)
	if err != nil {
		return nil, err
	}
	var b Block
	if err := json.Unmarshal(raw, &b); err != nil {
		return nil, err
	}
	// integrity: recompute header hash
	expect := hashHeader(b.Header)
	if b.Hash != expect && b.Hash != "0xGENESIS" {
		return nil, fmt.Errorf("hash mismatch: have %s want %s", b.Hash, expect)
	}
	return &b, nil
}

// LoadBlock reads and verifies a block from an exact path (either index).
func LoadBlock(path string) (*Block, error) {
	return loadBlockFromPath(path)
}

func loadBlockFromPath(path string) (*Block, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var b Block
	if err := json.Unmarshal(raw, &b); err != nil {
		return nil, err
	}
	// integrity: recompute header hash
	expect := hashHeader(b.Header)
	if b.Hash != expect && b.Hash != "0xGENESIS" {
		return nil, fmt.Errorf("hash mismatch: have %s want %s", b.Hash, expect)
	}
	return &b, nil
}

// ------------------------------- Utilities --------------------------------

func NowUTC() int64 { return time.Now().UTC().Unix() }

// MineNextFull: konstruksi & persist block (by-hash full, by-unix hash-only)
func MineNextFull(parent *Block, genesisTime int64, roots Roots, extra string) (*Block, string, error) {
	ts := NowUTC()
	if ts < parent.Header.Timestamp {
		ts = parent.Header.Timestamp
	}
	nb, err := NewBlockFull(parent, ts, genesisTime, roots, extra)
	if err != nil {
		return nil, "", err
	}
	_, hashPath, err := SaveBlockSimple(nb)
	if err != nil {
		return nil, "", err
	}
	return nb, hashPath, nil
}

func sanitizeHash(h Hash) string {
	if len(h) >= 2 && h[:2] == "0x" {
		return h[2:]
	}
	return h
}
