package main

import (
	"github.com/megalypse/go-svc-cluster/cmd/cli"
	"github.com/megalypse/go-svc-cluster/internal/commands"
)

func main() {
	cli.RootCmd.AddCommand(commands.CmdNewCluster)
	cli.Execute()
}
