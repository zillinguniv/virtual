package node

import (
	"log"
	"os"
	"time"

	"virtual/core"
)

// InitChain men-setup folder data, config genesis, lalu menentukan head awal.
// Menyimpan genesisTime ke field n.genesisTime agar service lain bisa pakai.
func (n *Node) InitChain() {
	ensureDir(n.cfg.DataDir)
	if err := os.Chdir(n.cfg.DataDir); err != nil {
		log.Fatalf("chdir: %v", err)
	}

	cfgPath := core.BlocksRootSimple + "/CONFIG.json"
	var onDiskCfg ConfigFile
	ok, err := readJSON(cfgPath, &onDiskCfg)
	if err != nil {
		log.Fatalf("read CONFIG: %v", err)
	}

	if !ok {
		if n.cfg.GenesisTs == 0 {
			n.cfg.GenesisTs = time.Now().UTC().Unix()
		}
		onDiskCfg = ConfigFile{GenesisTime: n.cfg.GenesisTs}
		if err := writeJSON(cfgPath, &onDiskCfg); err != nil {
			log.Fatalf("write CONFIG: %v", err)
		}
		log.Printf("init CONFIG genesisTime=%d (%s)",
			onDiskCfg.GenesisTime, time.Unix(onDiskCfg.GenesisTime, 0).UTC(),
		)
	}
	n.genesisTime = onDiskCfg.GenesisTime

	// coba load genesis via by-unix/0.json
	g, err := core.LoadBlockByUnix(0)
	if err != nil {
		// belum ada: buat genesis
		gen := core.NewGenesisFull(onDiskCfg.GenesisTime, n.cfg.ExtraData, core.Roots{}, true)
		up, hp, err := core.SaveBlockSimple(gen)
		if err != nil {
			log.Fatalf("save genesis: %v", err)
		}
		n.head = gen
		n.headBy = HeadFile{
			Hash:      gen.Hash,
			Timestamp: gen.Header.Timestamp,
			VTCUnix:   gen.Header.VTCUnix,
			PathUnix:  up,
			PathHash:  hp,
		}
		_ = writeJSON(core.BlocksRootSimple+"/HEAD.json", &n.headBy)
		log.Printf("genesis created hash=%s unix=%d", gen.Hash, gen.Header.VTCUnix)
		return
	}

	// minimal sudah ada genesis: coba HEAD.json agar akurat
	var hf HeadFile
	if ok, _ := readJSON(core.BlocksRootSimple+"/HEAD.json", &hf); ok {
		if b, err := core.LoadBlock(hf.PathHash); err == nil {
			n.head = b
			n.headBy = hf
		} else {
			n.head = g
			n.headBy = HeadFile{
				Hash:      g.Hash,
				Timestamp: g.Header.Timestamp,
				VTCUnix:   g.Header.VTCUnix,
			}
		}
	} else {
		n.head = g
		n.headBy = HeadFile{
			Hash:      g.Hash,
			Timestamp: g.Header.Timestamp,
			VTCUnix:   g.Header.VTCUnix,
		}
	}
	log.Printf("loaded head hash=%s unix=%d", n.head.Hash, n.head.Header.VTCUnix)
}
