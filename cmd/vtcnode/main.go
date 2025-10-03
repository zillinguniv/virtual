package main

import (
	"flag"
	"log"
	"strings"
	"time"

	"virtual/internal/node"
)

func main() {
	var (
		datadir   = flag.String("datadir", ".", "data directory")
		genesis   = flag.Int64("genesis", 0, "genesis UNIX time (UTC), default now if not set")
		interval  = flag.Duration("interval", 2*time.Second, "mining interval, 0=disable mining")
		p2plisten = flag.String("p2p.listen", ":30333", "p2p listen address")
		peers     = flag.String("peers", "", "comma-separated seed peers host:port")
		rpclisten = flag.String("rpc.listen", ":8545", "rpc http listen address")
		extra     = flag.String("extra", "vtcnode", "extraData for header")
	)
	flag.Parse()

	cfg := node.NodeConfig{
		DataDir:   *datadir,
		GenesisTs: *genesis,
		MineEvery: *interval,
		P2PListen: *p2plisten,
		RPCListen: *rpclisten,
		ExtraData: *extra,
	}
	if *peers != "" {
		cfg.PeerSeeds = strings.Split(*peers, ",")
	}

	n := node.New(cfg)
	n.InitChain()
	log.Printf("genesisTime=%d", n.GenesisTime())

	n.StartP2P()
	n.StartMining()
	n.StartRPC()

	select {} // block forever
}

