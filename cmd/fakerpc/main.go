package main

import (
	"os"

	"github.com/rjeczalik/fakerpc/cli"
)

func main() {
	cli.NewCLI().Run(os.Args)
}
