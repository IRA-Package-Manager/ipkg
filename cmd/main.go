package main

import (
	"fmt"
	"os"

	cmd "github.com/ira-package-manager/ipkg/cmd/lib"
)

func main() {
	err := cmd.RunSubcommand(
		[]cmd.Interface{
			cmd.NewInstallCommand(),
			cmd.NewOpenRootCommand(),
		}, os.Args[1:])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
