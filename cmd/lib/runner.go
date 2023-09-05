package cmd

import (
	"errors"
	"fmt"
)

func RunSubcommand(cmds []Interface, args []string) error {
	if len(args) < 1 {
		return ErrNoSubcommand
	}

	subcommand := args[1]
	for _, cmd := range cmds {
		if cmd.Name() == subcommand {
			err := cmd.Init(args[2:])
			if err != nil {
				return fmt.Errorf("initialising command: %w", err)
			}
			return cmd.Run()
		}
	}
	return fmt.Errorf("unknown subcommand: %s", subcommand)
}

var ErrNoSubcommand = errors.New("you must pass a sub-command")
