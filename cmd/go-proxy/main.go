package main

import (
	"go-proxy/internal/cmd"
	"go-proxy/internal/config"
)

//nolint:gochecknoglobals
var (
	Version     = "" //  "1.0.0"
	ShortCommit = "" // "1a2b3c4"
	Commit      = "" // "1a2b3c4d5e6f7g8h9i0j1k2l3m4n5o6p7q8r9s0"
	Date        = ""
)

func main() {

	config.AppVersion, config.AppCommit, config.AppDate, config.ShortCommit = Version, Commit, Date, ShortCommit

	config.ReadFlags()
	//
	x := cmd.Command{}

	x.Exec()
}
