package main

import (
	"fmt"
	"os"

	"github.com/ira-package-manager/gobetter/cmd"
)

func main() {
	err := cmd.RunSubcommand(
		[]cmd.Interface{
			NewInstallCommand(),
			NewOpenRootCommand(),
		}, os.Args[1:])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
