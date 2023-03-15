package ctrler

import (
	"podspider/ctrler/resource"
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
	log.Debug("ClusterCtrler Ticking")
	ctrler.resource_mgr.Tick()
}
