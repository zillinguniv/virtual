package node

import (
	"log"
	"time"

	"virtual/core"
)

func (n *Node) StartMining() {
	if n.cfg.MineEvery <= 0 {
		return
	}
	ticker := time.NewTicker(n.cfg.MineEvery)
	go func() {
		for range ticker.C {
			n.mu.Lock()
			parent := n.head
			nb, _, err := core.MineNextFull(parent, n.genesisTime, core.Roots{}, n.cfg.ExtraData)
			if err != nil {
				n.mu.Unlock()
				log.Printf("mine error: %v", err)
				continue
			}
			up, hp, _ := core.SaveBlockSimple(nb)
			n.head = nb
			n.headBy = HeadFile{
				Hash:      nb.Hash,
				Timestamp: nb.Header.Timestamp,
				VTCUnix:   nb.Header.VTCUnix,
				PathUnix:  up,
				PathHash:  hp,
			}
			_ = writeJSON(core.BlocksRootSimple+"/HEAD.json", &n.headBy)
			n.mu.Unlock()

			log.Printf("mined height~%d unix=%d hash=%s", nb.Header.VTCUnix, nb.Header.VTCUnix, nb.Hash)
			n.broadcastAnnounce(nb)
		}
	}()
}
