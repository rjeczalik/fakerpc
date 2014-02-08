package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"net/url"
	"os"
	"strings"
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

var session io.ReadWriteCloser

var (
	errFast  = errors.New("bp: -fast cannot be used without -server or -client")
	errHttps = errors.New("bp cannot into https :( [https://github.com/rjecza" +
		"lik/blacktarget/issues/1")
	errTimedReplyNotImpl = errors.New("bp: -fast=false not implemented :( " +
		"https://github.com/rjeczalik/blacktarget/issues/2")
	errSessionFile = errors.New("bp: expected session file path as a single" +
		" argument")
	errRecordReply = errors.New("bp: -target cannot be mixed with -fast, -se" +
		"rver or -client")
	errServerClient = errors.New("bp: one of -server or -client is required")
	errAmbiguous    = errors.New("bp: only one of -server and -client can be " +
		"specified at once")
)

func init() {
	flag.StringVar(&addr, "addr", ":8080", "")
	flag.StringVar(&target, "target", "", "")
	flag.BoolVar(&server, "server", false, "")
	flag.BoolVar(&client, "client", false, "")
	// TODO reply time diff between request when -fast=false
	flag.BoolVar(&fast, "fast", false, "")
	flag.BoolVar(&help, "help", false, "")
	flag.Parse()
}

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
	if addr, err = completeURL(addr); err != nil {
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

func main() {
	if help {
		fmt.Println(`usage: bp [-help | [[RECORD OPTIONS | REPLY OPTIONS] session_file]

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
`)
		return
	}
	if err := validate(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	// TODO
}
