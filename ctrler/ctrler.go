package ctrler

import (
	"dspm/ctrler/resource"
	"time"

	log "github.com/sirupsen/logrus"
)

type ClusterCtrler struct {
	config       CtrlerConfig
	ticker       *time.Ticker
	resource_mgr resource.ResourceManager
}

func MakeClusterCtrler(config CtrlerConfig) *ClusterCtrler {
	ctrler := new(ClusterCtrler)
	ctrler.config = config
	ctrler.ticker = time.NewTicker(time.Duration(config.Tick_interval) * time.Second)

	return ctrler
}

func (ctrler *ClusterCtrler) Start() {
	go ctrler.resource_mgr.Start(ctrler.config.WatchedNs)

	for {
		<-ctrler.ticker.C
		ctrler.tick()
	}
}

func (ctrler *ClusterCtrler) tick() {
	// 1. gather cluster information
	// 2. make resource decision
	// 3. make migration decision
	log.Info("ClusterCtrler Ticking")
	ctrler.resource_mgr.Tick()
}
