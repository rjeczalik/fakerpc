// Fakerpc is a command line interface to the fakerpc server.
//
// Fakerpc has two modes - proxy mode and fake mode. In the proxy mode fakerpc sets up
// a recording proxy server which listens on the address specified by the --addr flag
// and records all the traffic to the file provided by the --log flag.
// The following example starts a server that listens on http://localhost:8079,
// proxying all the requests to http://github.com and recording all the requests
// and responses:
//
//   $ fakerpc --addr localhost:8079 record http://github.com
//   fakerpc: Proxy recording on 127.0.0.1:8079 . . .
//
// This clones a repository via fakerpc proxy address:
//
//   $ git clone http://localhost:8079/rjeczalik/fakerpc.git
//
// Sending SIGINT fo the fakerpc stops it from recording and saves the transmission
// history to the log file:
//
//   $ sudo ./fakerpc --addr localhost:8079 record https://github.com
//   fakerpc: Proxy recording on 172.17.42.1:80 . . .
//   fakerpc: T 172.0.0.1:49740 -> 192.30.252.129:80 (169)
//   fakerpc: T 192.30.252.129:80 -> 172.0.0.1:49740 (217)
//   ^Cfakerpc: Signal caught; stopping proxy . . .
//   fakerpc: Saving log to the "/home/rjeczalik/fakerpc.gzob.1" file . . .
//
// The second mode allows for fakerpc acting as a actual fake server - it listens
// on an address provided by the --addr flag and reads a log file specified by
// the --log flag. Example:
//
//   fakerpc --addr localhost:8079 --log /home/rjeczalik/fakerpc.gzob.1 reply
//
// After cloning a repository from the fake, the server itself will shutdown as
// soon as the transmission is completed.
//
// Usage:
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
