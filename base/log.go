package base

import (
	"github.com/hhq163/logger"
)

var Log logger.Logger

func LogInit(debug bool, svcName string) {
	if debug {
		cfg := logger.NewDevelopmentConfig()
		cfg.Encoding = "json"
		cfg.OutputPaths = append(cfg.OutputPaths, "access_log.txt")
		Log = logger.NewMyLogger(cfg)

		// cfg.Filename = "./access_log.txt"
		// cfg.MaxSize = 100 //单位为M
		// Log = logger.NewCuttingLogger(cfg)

	} else {
		cfg := logger.NewProductionConfig()
		Log = logger.NewMyLogger(cfg)

		// cfg.Filename = "./access_log.txt"
		// cfg.MaxSize = 100 //单位为M
		// Log = logger.NewCuttingLogger(cfg)
	}

}
