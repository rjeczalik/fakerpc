package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/url"
	"os"
	"os/signal"
	"strings"

	"github.com/rjeczalik/blackproxy"
)

var (
	addr   string
	target string
	server bool
	client bool
	check  bool
	fast   bool
	help   bool
)

var (
	session io.ReadWriteCloser
	cleanup = make(chan func() error, 1)
)

var (
	errFast  = errors.New("-fast cannot be used without -server or -client")
	errHttps = errors.New("cannot into https :( [https://github.com/rjecza" +
		"lik/blackproxy/issues/1]")
	errTimedReplyNotImpl = errors.New("-fast=false not implemented :( " +
		"[https://github.com/rjeczalik/blackproxy/issues/2]")
	errSessionFile = errors.New("expected session file path as a single" +
		" argument")
	errRecordReply = errors.New("-target cannot be mixed with -fast, -se" +
		"rver or -client")
	errServerClient = errors.New("one of -server or -client is required")
	errAmbiguous    = errors.New("only one of -server and -client can be " +
		"specified at once")
)

const usage = `usage: bp [-help | [[RECORD OPTIONS | REPLY OPTIONS] session_file]

  session_file  requests are written to this file when recording or read when
                replying

  -help         print usage information

RECORD OPTIONS:

  -addr    HTTP reverse proxy address (default is http://localhost:8080)

  -target  HTTP target service address

REPLY OPTIONS:

  -addr    HTTP service address

  -client  reply session acting as a client; requests are sent to -addr

  -server  reply session acting as a server; server listens on -addr

  -fast    issues requests immadiately without replying time diff between them
`

func completeURL(addr string) (u string, err error) {
	if strings.HasPrefix(addr, "https://") {
		err = errHttps
		return
	}
	u = addr
	if !strings.HasPrefix(u, "http://") {
		if !strings.Contains(u, ":") {
			u = u + ":80"
		}
		host, port, err := net.SplitHostPort(u)
		if err != nil {
			return "", err
		}
		if host == "" {
			host = "localhost"
		}
		u = "http://" + host
		if port != "" {
			u = u + ":" + port
		}
	}
	_, err = url.Parse(u)
	return
}

func validate() (err error) {
	if flag.NArg() != 1 {
		return errSessionFile
	}
	if fast && !server && !client {
		return errFast
	}
	if server && client {
		return errAmbiguous
	}
	if _, _, err = net.SplitHostPort(addr); err != nil {
		return
	}
	if target != "" {
		if server || client || fast {
			return errRecordReply
		}
		if target, err = completeURL(target); err != nil {
			return
		}
	} else if !server && !client {
		return errServerClient
	}
	if server || client {
		session, err = os.Open(flag.Arg(0))
	} else {
		session, err = os.OpenFile(flag.Arg(0), os.O_WRONLY|os.O_CREATE, 0644)
	}
	return
}

func sighandler(ch <-chan os.Signal) {
	if _, ok := <-ch; ok {
		select {
		case clfn, ok := <-cleanup:
			if !ok {
				return
			}
			log.Println("Interrupted, cleaning up . . .")
			if err := clfn(); err == nil {
				// TODO push interrupt.Server to rjeczalik/netutil and use it
				//      instead of http.ListenAndServe; this is crucial to stop
				//      HTTP server first and then allow dumper to finish gracefully
				os.Exit(0)
			} else {
				fatal(err)
			}
		default:
			os.Exit(0)
		}
	}
}

func fatal(err error) {
	fmt.Fprintf(os.Stderr, "bp: %s\n", err)
	os.Exit(1)
}

func init() {
	// interrupt handling
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt, os.Kill)
	go sighandler(ch)
	// command line handling
	flag.StringVar(&addr, "addr", ":8080", "")
	flag.StringVar(&target, "target", "", "")
	flag.BoolVar(&server, "server", false, "")
	flag.BoolVar(&client, "client", false, "")
	flag.BoolVar(&fast, "fast", false, "")
	flag.BoolVar(&help, "help", false, "")
	flag.Parse()
}

func main() {
	if help {
		fmt.Println(usage)
		return
	}
	if err := validate(); err != nil {
		fatal(err)
	}
	// record mode
	if target != "" {
		rp := black.NewRecordingProxy(session)
		cleanup <- func() error { return rp.Close() }
		log.Println("Recording proxy session", addr, "<->", target, "to", flag.Arg(0), ". . .")
		if err := rp.ListenAndServe(addr, target); err != nil {
			fatal(err)
		}
	}
	// reply mode
	switch {
	case server:
		// TODO
	case client:
		// TODO
	}
}
