package cli

import (
	"bytes"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"os/user"
	"path/filepath"

	"github.com/rjeczalik/fakerpc"

	"github.com/codegangsta/cli"
)

var defaultErr = func(args ...interface{}) {
	for _, arg := range args {
		fmt.Fprintln(os.Stderr, arg)
	}
}

var defaultOut = func(args ...interface{}) {
	for _, arg := range args {
		fmt.Println(arg)
	}
}

func logfile() (path string) {
	u, err := user.Current()
	if err != nil {
		log.Fatal(err)
	}
	base := filepath.Join(u.HomeDir, "fakerpc.gzob")
	for i := 0; i < 1024; i++ {
		path = fmt.Sprintf("%s.%d", base, i)
		if _, err = os.Stat(path); os.IsNotExist(err) {
			return
		}
	}
	return ""
}

// CLI TODO(rjeczalik): document
type CLI struct {
	Out  func(...interface{})
	Err  func(...interface{})
	Exit func(int)
	app  *cli.App
}

func NewCLI() *CLI {
	cl := &CLI{
		Out:  defaultOut,
		Err:  defaultErr,
		Exit: os.Exit,
		app:  cli.NewApp(),
	}
	cl.app.Name = "fakerpc"
	cl.app.Version = "0.1.0"
	cl.app.Usage = "use gentle and with great care"
	cl.app.Flags = []cli.Flag{
		cli.StringFlag{Name: "addr", Value: "localhost:0", Usage: "An address to listen on"},
		cli.StringFlag{Name: "log", Value: logfile(), Usage: "A path to the record-log file (or ngrep output)"},
	}
	cl.app.Commands = []cli.Command{{
		Name:   "record",
		Usage:  "Proxies connections recording them all to the record-log",
		Action: cl.Record,
	}, {
		Name:   "reply",
		Usage:  "Serves connections with recorded responses from the record-log",
		Action: cl.Reply,
	}, {
		Name:   "show",
		Usage:  "Shows record-log as a ngrep output",
		Action: cl.Show,
	}}
	return cl
}

// Proxy TODO(rjeczalik): document
func (cl *CLI) Record(ctx *cli.Context) {
	target := ctx.Args().First()
	if target == "" {
		cl.Err("fakerpc: missing (...) record <proxy target url>")
		cl.Exit(1)
	}
	p, err := fakerpc.NewProxy(ctx.GlobalString("addr"), target)
	if err != nil {
		cl.Err(err)
		cl.Exit(1)
	}
	p.Record = func(t *fakerpc.Transmission) {
		cl.Out(fmt.Sprintf("fakerpc: T %s -> %s (%d)", t.Src, t.Dst, len(t.Raw)))
	}
	done, sig := make(chan struct{}), make(chan os.Signal, 1)
	go func() {
		if err := p.ListenAndServe(); err != nil {
			cl.Err(err)
			cl.Exit(1)
		}
		close(done)
	}()
	signal.Notify(sig, os.Interrupt, os.Kill)
	cl.Out(fmt.Sprintf("fakerpc: Proxy recording on %s . . .", p.Addr()))
	<-sig
	cl.Out("fakerpc: Signal caught; stopping proxy . . .")
	log, err := p.Stop()
	if err != nil {
		cl.Err(err)
		cl.Exit(1)
	}
	<-done
	logFile := ctx.GlobalString("log")
	cl.Out(fmt.Sprintf("fakerpc: Saving log to the %q file . . .", logFile))
	if err = fakerpc.WriteLog(logFile, log); err != nil {
		cl.Err(err)
		cl.Exit(1)
	}
}

// Server TODO(rjeczalik): document
func (cl *CLI) Reply(ctx *cli.Context) {
	l, err := fakerpc.ReadLog(ctx.GlobalString("log"))
	if err != nil {
		cl.Err(err)
		cl.Exit(1)
	}
	srv, err := fakerpc.NewServer(ctx.GlobalString("addr"), l)
	if err != nil {
		cl.Err(err)
		cl.Exit(1)
	}
	srv.Reply = func(src, dst *net.TCPAddr, n int64, err error) {
		if err != nil {
			cl.Err(fmt.Sprintf("fakerpc: T %s -> %s (%d) error: %v", src, dst, n, err))
		} else {
			cl.Out(fmt.Sprintf("fakerpc: T %s -> %s (%d)", src, dst, n))
		}
	}
	done := make(chan struct{})
	go func() {
		if err := srv.ListenAndServe(); err != nil {
			cl.Err(err)
			cl.Exit(1)
		}
		close(done)
	}()
	cl.Out(fmt.Sprintf("fakerpc: Server replying on %s . . .", srv.Addr()))
	<-done
}

// Show TODO(rjeczalik): document
func (cl *CLI) Show(ctx *cli.Context) {
	l, err := fakerpc.ReadLog(ctx.GlobalString("log"))
	if err != nil {
		cl.Err(err)
		cl.Exit(1)
	}
	var buf bytes.Buffer
	if err = fakerpc.NgrepMarshal(&buf, l); err != nil {
		cl.Err(err)
		cl.Exit(1)
	}
	cl.Out(buf.String())
}

// Run TODO(rjeczalik): document
func (cl *CLI) Run(args []string) {
	cl.app.Run(args)
}
