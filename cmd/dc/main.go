package main

import (
	"os"

	"github.com/Codezilla-jpg/plaincord/internal/cli"
)

var version = "0.2.0"

func main() {
	if version != "" {
		cli.Version = version
	}
	os.Exit(cli.Run(os.Args, os.Stdin, os.Stdout, os.Stderr))
}
