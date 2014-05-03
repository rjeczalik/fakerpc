// Usage
//
//   NAME:
//      fakerpc - use gentle and with great care
//
//   USAGE:
//      fakerpc [global options] command [command options] [arguments...]
//
//   VERSION:
//      0.1.0
//
//   COMMANDS:
//      record       Proxies connections recording them all to the record-log
//      reply        Serves connections with recorded responses from the record-log
//      show         Shows record-log as a ngrep output
//      help, h      Shows a list of commands or help for one command
//
//   GLOBAL OPTIONS:
//      --addr 'localhost:0'              An address to listen on
//      --log '${HOME}/fakerpc.gzob.0'    A path to the record-log file (or ngrep output)
//      --version, -v                     print the version
//      --help, -h                        show help
package main

import (
	"os"

	"github.com/rjeczalik/fakerpc/cli"
)

func main() {
	cli.NewCLI().Run(os.Args)
}
