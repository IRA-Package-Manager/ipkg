package cmd

import "errors"

type Interface interface {
	Name() string
	Run() error // If Init() wasn't executed, drops ErrNotReady
	Init([]string) error
}

var ErrNotReady = errors.New("cmd.Interface not ready. Must execute cmd.Interface.Init() before running")
