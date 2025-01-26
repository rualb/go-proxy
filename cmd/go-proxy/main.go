package main

import (
	"cmp"
	_ "embed"
	"go-proxy/internal/cmd"
	"go-proxy/internal/config"

	"go-proxy/internal/config/consts"
	xlog "go-proxy/internal/util/utillog"
)

//go:embed date
var date string

//nolint:gochecknoglobals
var (
	Version     = "" //  "1.0.0"
	ShortCommit = "" // "1a2b3c4"
	Commit      = "" // "1a2b3c4d5e6f7g8h9i0j1k2l3m4n5o6p7q8r9s0"
	Date        = ""
)

func main() {

	xlog.Info("build info: [name: %v] [version: %v] [date: %v] [short-commit: %v]", consts.AppName, Version, cmp.Or(Date, date), ShortCommit)

	config.AppVersion, config.AppCommit, config.AppDate, config.ShortCommit = Version, Commit, Date, ShortCommit

	config.ReadFlags()
	//
	x := cmd.Command{}

	x.Exec()
}
