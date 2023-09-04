package cmd

import (
	"flag"
	"os"

	"github.com/fatih/color"
	"github.com/ira-package-manager/ipkg"
)

type OpenRoot struct {
	flagSet *flag.FlagSet
	ready   bool
	path    string
}

func NewOpenRootCommand() *OpenRoot {
	return &OpenRoot{
		flagSet: flag.NewFlagSet("root", flag.ContinueOnError),
		ready:   false,
	}
}

func (or *OpenRoot) Init(args []string) error {
	err := or.flagSet.Parse(args)
	if err != nil {
		return err
	}
	or.path = or.flagSet.Arg(0)
	or.ready = true
	return nil
}

func (or *OpenRoot) Name() string { return or.flagSet.Name() }

func (or *OpenRoot) Run() error {
	if !or.ready {
		return ErrNotReady
	}
	if _, err := os.Stat(or.path); os.IsNotExist(err) {
		root, err := ipkg.CreateRoot(or.path)
		if err != nil {
			return err
		}
		color.Green("Root %s succesifully created", or.path)
		config.root = root
	} else {
		root, err := ipkg.CreateRoot(or.path)
		if err != nil {
			return err
		}
		color.Green("Root %s succesifully opened", or.path)
		config.root = root
	}
	return nil
}
