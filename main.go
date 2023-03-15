package main

import (
	"os"

	log "github.com/sirupsen/logrus"

	"podspider/ctrler"
)

func logger_init(config ctrler.CtrlerConfig) {
	switch config.Log_level {
	case "debug":
		log.SetLevel(log.DebugLevel)
	case "info":
		log.SetLevel(log.InfoLevel)
	case "warn":
		log.SetLevel(log.WarnLevel)
	case "error":
		log.SetLevel(log.ErrorLevel)
	case "fatal":
		log.SetLevel(log.FatalLevel)
	case "panic":
		log.SetLevel(log.PanicLevel)
	default:
		log.SetLevel(log.InfoLevel)
	}

	log.SetOutput(os.Stdout)
}

func main() {
	config := ctrler.NewCtrlerConfig("./ps_config.toml")
	logger_init(config)

	config.Tick_interval = 1
	ctrler := ctrler.MakeClusterCtrler(config)

	ctrler.Start()
}
