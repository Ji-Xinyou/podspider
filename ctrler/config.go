package ctrler

import (
	"os"

	toml "github.com/BurntSushi/toml"
	log "github.com/sirupsen/logrus"
)

// the configuration of clusterctrler, all fields should be cap
type CtrlerConfig struct {
	// the build of dspm
	Build_mode string

	// the interval that the ctrler ticks and make executions
	Tick_interval int

	// the namespace filter this ctrler will watch
	WatchedNs string
}

func DefaultCtrlerConfig() CtrlerConfig {
	return CtrlerConfig{
		Build_mode:    "debug",
		Tick_interval: 5,
		WatchedNs:     "",
	}
}

// parse ctrler config from TOML file
func NewCtrlerConfig(path string) CtrlerConfig {
	if _, err := os.Stat(path); err != nil {
		return DefaultCtrlerConfig()
	}

	var config CtrlerConfig
	_, err := toml.DecodeFile(path, &config)
	if err != nil {
		log.Fatal("TOML config not valid %s", err)
	}

	log.Debugf("%s", config)

	return config
}
