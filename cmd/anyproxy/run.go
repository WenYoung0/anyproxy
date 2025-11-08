package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/wenyoung0/anyproxy/config"

	"github.com/qtraffics/qtfra/log"

	"github.com/goccy/go-yaml"
	"github.com/spf13/cobra"
	"github.com/wenyoung0/anyproxy"
)

var runCommand = &cobra.Command{
	Use:   "run",
	Short: "Run server",
	Run:   run,
}

var configPath string

func init() {
	defer rootCommand.AddCommand(runCommand)
	runCommand.Flags().StringVarP(&configPath, "config", "c", "config.yml", "Set Config file path")
}

func run(cmd *cobra.Command, args []string) {
	logger := log.GetDefaultLogger()
	c, err := readConfig()
	if err != nil {
		logger.Error("readConfig failed!", log.AttrError(err))
		return
	}
	proxy, err := anyproxy.NewProxy(c)
	if err != nil {
		logger.Error("Build server failed!", log.AttrError(err))
		return
	}
	rootCtx, cancel := context.WithCancel(context.Background())
	defer cancel()
	wg := &sync.WaitGroup{}

	err = proxy.Serve(wg, rootCtx)
	if err != nil {
		logger.Error("Start server failed!", log.AttrError(err))
		return
	}

	sigChannel := make(chan os.Signal, 1)
	signal.Notify(sigChannel, os.Interrupt, os.Kill, syscall.SIGHUP)

	select {
	case <-rootCtx.Done():
	case sig := <-sigChannel:
		logger.Warn("Signal received", slog.String("signal", sig.String()))
		cancel()
	}

	wg.Wait()
}

func readConfig() (config.Config, error) {
	cc := config.Default()
	file, err := os.Open(configPath)
	if err != nil {
		return cc, err
	}
	defer file.Close()

	decoder := yaml.NewDecoder(file, yaml.DisallowUnknownField())
	err = decoder.Decode(&cc)
	return cc, err
}
