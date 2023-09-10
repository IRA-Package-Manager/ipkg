package main

import (
	"flag"
	"os"
	"path/filepath"

	"github.com/fatih/color"
	"github.com/ira-package-manager/gobetter/cmd"
	"github.com/ira-package-manager/ipkg"
)

type Install struct {
	flagSet      *flag.FlagSet
	ready        bool
	path         string
	asDependency bool
}

func NewInstallCommand() *Install {
	install := &Install{
		flagSet: flag.NewFlagSet("install", flag.ContinueOnError),
		ready:   false,
	}
	install.flagSet.BoolVar(&install.asDependency, "dependency", false, "If specified, package will be installed as dependency")

	return install
}

func (i *Install) Init(args []string) error {
	err := i.flagSet.Parse(args)
	if err != nil {
		return err
	}
	i.path = i.flagSet.Arg(0)
	i.ready = true
	return nil
}

func (i *Install) Name() string {
	return i.flagSet.Name()
}

func (i *Install) Run() error {
	if !i.ready {
		return cmd.ErrNotReady
	}
	if config.root == nil {
		home, err := os.UserHomeDir()
		if err != nil {
			return err
		}
		if _, err := os.Stat(filepath.Join(home, ".ira", "db.sqlite3")); os.IsNotExist(err) {
			config.root, err = ipkg.CreateRoot(filepath.Join(home, ".ira", "db.sqlite3"))
			if err != nil {
				return err
			}
		} else {
			config.root, err = ipkg.OpenRoot(filepath.Join(home, ".ira", "db.sqlite3"))
			if err != nil {
				return err
			}
		}
	}
	err := config.root.InstallPackage(i.path, i.asDependency)
	if err != nil {
		return err
	}
	color.Green("Package %s succesifully installed", i.path)
	return nil
}
