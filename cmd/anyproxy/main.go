package main

import (
	"os"

	"github.com/spf13/cobra"
)

var rootCommand = &cobra.Command{
	Use:   os.Args[0],
	Short: "Proxy any url as u needed(like enhanced ghproxy)",
}

func main() {
	err := rootCommand.Execute()
	if err != nil {
		panic(err)
	}
}
