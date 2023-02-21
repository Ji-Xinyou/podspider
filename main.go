package main

import (
	"os"

	log "github.com/sirupsen/logrus"

	"dspm/ctrler"
)

func logger_init(env string) {
	if env == "production" {
		// prod mode, output to stderr
		log.SetFormatter(&log.JSONFormatter{})
		log.SetLevel(log.InfoLevel)
	} else {
		// debug mode, output to stdout
		log.SetLevel(log.DebugLevel)
		log.SetOutput(os.Stdout)
	}
}

func main() {
	config := ctrler.NewCtrlerConfig("/home/xyji/dspm_config.toml")
	logger_init(config.Build_mode)

	config.Tick_interval = 1
	ctrler := ctrler.MakeClusterCtrler(config)

	ctrler.Start()
}
