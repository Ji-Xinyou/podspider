package main

import (
	"os"

	log "github.com/sirupsen/logrus"

	"podspider/ctrler"
)

func logger_init(config ctrler.CtrlerConfig) {
	logLevels := map[string]log.Level{
		"debug": log.DebugLevel,
		"info":  log.InfoLevel,
		"warn":  log.WarnLevel,
		"error": log.ErrorLevel,
		"fatal": log.FatalLevel,
		"panic": log.PanicLevel,
	}

	level, ok := logLevels[config.Log_level]
	if !ok {
		level = log.InfoLevel
	}

	log.SetLevel(level)

	log.SetOutput(os.Stdout)
}

func main() {
	config := ctrler.NewCtrlerConfig("./ps_config.toml")
	logger_init(config)

	config.Tick_interval = 1
	ctrler := ctrler.MakeClusterCtrler(config)

	ctrler.Start()
}
