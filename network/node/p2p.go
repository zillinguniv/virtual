package node

import (
	"bufio"
	"encoding/json"
	"log"
	"net"
	"strings"
	"time"

	"virtual/core"
)

func (n *Node) StartP2P() {
	if n.cfg.P2PListen != "" {
		go n.listenLoop(n.cfg.P2PListen)
	}
	for _, addr := range n.cfg.PeerSeeds {
		if addr == "" {
			continue
		}
		go n.dialLoop(strings.TrimSpace(addr))
	}
}

func (n *Node) listenLoop(addr string) {
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalf("p2p listen %s: %v", addr, err)
	}
	log.Printf("P2P listening on %s", addr)
	for {
		c, err := ln.Accept()
		if err != nil {
			log.Printf("accept: %v", err)
			continue
		}
		n.registerPeer(c)
		go n.handleConn(c)
	}
}

func (n *Node) dialLoop(addr string) {
	for {
		c, err := net.Dial("tcp", addr)
		if err != nil {
			time.Sleep(2 * time.Second)
			continue
		}
		n.registerPeer(c)
		go n.handleConn(c)
		return
	}
}

func (n *Node) registerPeer(c net.Conn) {
	n.mu.Lock()
	n.peers[c.RemoteAddr().String()] = c
	n.mu.Unlock()

	// kirim head announce segera
	n.mu.RLock()
	head := n.head
	n.mu.RUnlock()
	if head != nil {
		sendJSON(c, WireMsg{Type: MsgAnnounce, Hash: head.Hash, VTCUnix: head.Header.VTCUnix})
	}
}

func (n *Node) handleConn(c net.Conn) {
	defer func() {
		n.mu.Lock()
		delete(n.peers, c.RemoteAddr().String())
		n.mu.Unlock()
		c.Close()
	}()

	rd := bufio.NewReader(c)
	for {
		line, err := rd.ReadBytes('\n')
		if err != nil {
			return
		}
		var m WireMsg
		if err := json.Unmarshal(line, &m); err != nil {
			continue
		}
		n.handleMsg(c, m)
	}
}

func (n *Node) handleMsg(c net.Conn, m WireMsg) {
	switch m.Type {
	case MsgAnnounce:
		n.mu.RLock()
		have := n.head != nil && (n.head.Hash == m.Hash || n.head.Header.VTCUnix >= m.VTCUnix)
		n.mu.RUnlock()
		if !have {
			sendJSON(c, WireMsg{Type: MsgGetBlock, WantHash: m.Hash})
		}

	case MsgGetBlock:
		b, err := core.LoadBlockByHash(m.WantHash)
		if err == nil {
			sendJSON(c, WireMsg{Type: MsgBlock, Block: b})
		}

	case MsgBlock:
		if m.Block == nil {
			return
		}
		n.mu.Lock()
		if n.head == nil || m.Block.Header.VTCUnix > n.head.Header.VTCUnix {
			up, hp, err := core.SaveBlockSimple(m.Block)
			if err == nil {
				n.head = m.Block
				n.headBy = HeadFile{
					Hash:      m.Block.Hash,
					Timestamp: m.Block.Header.Timestamp,
					VTCUnix:   m.Block.Header.VTCUnix,
					PathUnix:  up,
					PathHash:  hp,
				}
				_ = writeJSON(core.BlocksRootSimple+"/HEAD.json", &n.headBy)
				log.Printf("accepted remote block unix=%d hash=%s", m.Block.Header.VTCUnix, m.Block.Hash)
				n.mu.Unlock()
				n.broadcastAnnounce(m.Block)
				return
			}
		}
		n.mu.Unlock()
	}
}

func (n *Node) broadcastAnnounce(b *core.Block) {
	n.mu.RLock()
	for _, c := range n.peers {
		sendJSON(c, WireMsg{Type: MsgAnnounce, Hash: b.Hash, VTCUnix: b.Header.VTCUnix})
	}
	n.mu.RUnlock()
}

func sendJSON(c net.Conn, v any) {
	b, _ := json.Marshal(v)
	c.Write(append(b, '\n'))
}
