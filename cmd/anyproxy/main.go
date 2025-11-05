package main

import (
	"fmt"
	"os"

	"github.com/wenyoung0/anyproxy/common"

	"github.com/qtraffics/qtfra/log"
	"github.com/qtraffics/qtfra/log/loghandler"
	"github.com/qtraffics/qtfra/sys/sysvars"

	"github.com/spf13/cobra"
)

var rootCommand = &cobra.Command{
	Use:              os.Args[0],
	Short:            "Proxy any url as u needed(like enhanced ghproxy)",
	PersistentPreRun: preRun,
}

func preRun(cmd *cobra.Command, args []string) {
	logLevel := log.LevelInfo

	if logLevelStr := os.Getenv("LOG_LEVEL"); len(logLevelStr) != 0 {
		err := logLevel.UnmarshalText([]byte(logLevelStr))
		if err != nil {
			fmt.Printf("Warning: Env LOG_LEVEL=%s is an invalid level , use Info as default level\n", logLevelStr)
			logLevel = log.LevelInfo
		}
	} else if sysvars.DebugEnabled {
		logLevel = log.LevelDebug
	}

	handler := common.Must(loghandler.New(loghandler.BuildOption{
		Level:        log.LevelInfo,
		Time:         false,
		Debug:        sysvars.DebugEnabled,
		OutputWriter: os.Stdout,
	}))

	logger := log.New(handler)
	log.SetDefaultLogger(logger)
}

func main() {
	err := rootCommand.Execute()
	if err != nil {
		panic(err)
	}
}
