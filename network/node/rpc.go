package node

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"virtual/core"
)

func (n *Node) StartRPC() {
	if n.cfg.RPCListen == "" {
		return
	}

	http.HandleFunc("/status", func(w http.ResponseWriter, r *http.Request) {
		n.mu.RLock()
		defer n.mu.RUnlock()
		type Peer struct{ Addr string }
		peers := make([]Peer, 0, len(n.peers))
		for a := range n.peers {
			peers = append(peers, Peer{Addr: a})
		}
		resp := map[string]any{
			"head":        n.headBy,
			"peers":       peers,
			"genesisTime": n.genesisTime,
			"uptime":      time.Now().UTC().Format(time.RFC3339),
		}
		_ = json.NewEncoder(w).Encode(resp)
	})

	http.HandleFunc("/head", func(w http.ResponseWriter, r *http.Request) {
		n.mu.RLock()
		defer n.mu.RUnlock()
		_ = json.NewEncoder(w).Encode(n.head)
	})

	http.HandleFunc("/block", func(w http.ResponseWriter, r *http.Request) {
		h := r.URL.Query().Get("hash")
		b, err := core.LoadBlockByHash(h)
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		_ = json.NewEncoder(w).Encode(b)
	})

	http.HandleFunc("/block_unix", func(w http.ResponseWriter, r *http.Request) {
		u := r.URL.Query().Get("u")
		if u == "" {
			http.Error(w, "missing u", http.StatusBadRequest)
			return
		}
		var uu uint64
		fmt.Sscanf(u, "%d", &uu)
		b, err := core.LoadBlockByUnix(uu)
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		_ = json.NewEncoder(w).Encode(b)
	})

	go func() {
		log.Printf("RPC listening on %s", n.cfg.RPCListen)
		log.Fatal(http.ListenAndServe(n.cfg.RPCListen, nil))
	}()
}
