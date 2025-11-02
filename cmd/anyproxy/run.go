package main

import "github.com/spf13/cobra"

var runCommand = &cobra.Command{
	Use:   "run",
	Short: "Run server",
	Run:   run,
}

var (
	configPath string
)

func init() {
	defer rootCommand.AddCommand(runCommand)
	runCommand.Flags().StringVarP(&configPath, "config", "c", "config.yml", "Set Config file path")

}

func run(cmd *cobra.Command, args []string) {

}
