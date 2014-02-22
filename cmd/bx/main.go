package main

import (
	"errors"
	"flag"
	"io"
	"log"
	"os"
	"os/signal"
	"sync"

	"github.com/rjeczalik/blackproxy"
)

const ver = "0.1"

var (
	addr   string
	client string
	file   *os.File
	help   bool
	server string
	proxy  string
)

func btou(b bool) uint8 {
	if b {
		return 1
	}
	return 0
}

func parse() (run black.RunCloser, err error) {
	flag.Parse()
	if flag.NArg() != 1 {
		return nil, errors.New("bx: missing path to the session file")
	}
	isproxy, isclient, isserver := proxy != "", client != "", server != ""
	switch s := btou(isproxy) + btou(isclient) + btou(isserver); true {
	case s == 0:
		return nil, errors.New("bx: missing or empty command")
	case s > 1:
		return nil, errors.New("bx: command may be specified only one at a time")
	}
	if addr != ":0" && !isproxy {
		log.Printf("ignoring -addr=%q", addr)
	}
	f := os.O_CREATE
	if isclient || isserver {
		f = os.O_RDONLY
	}
	if file, err = os.OpenFile(flag.Arg(0), f, 0644); err != nil {
		return
	}
	switch {
	case isproxy:
		run, err = black.NewProxy(proxy, addr, file)
	case isclient:
		run, err = black.NewClient(client, file)
	case isserver:
		run, err = black.NewServer(server, file)
	}
	return
}

func handlesig(c io.Closer) {
	ch, once := make(chan os.Signal, 1), sync.Once{}
	signal.Notify(ch, os.Interrupt, os.Kill)
	go func() {
		for _ = range ch {
			once.Do(func() {
				if err := c.Close; err != nil {
					log.Println(err)
				}
			})
		}
	}()
}

func init() {
	flag.StringVar(&proxy, "proxy", "", "Proxies all requests to the target")
	flag.StringVar(&client, "client", "", "Replays requests connecting to given address")
	flag.StringVar(&server, "server", "", "Replays responses listening on given address")
	flag.StringVar(&addr, "addr", ":0", "Reverse proxy endpoint address")
}

func main() {
	run, err := parse()
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("bx v%s starting up . . .", ver)
	handlesig(run)
	if err = run.Run(); err != nil {
		log.Fatal(err)
	}
}
