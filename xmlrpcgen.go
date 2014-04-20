package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"os"
	"regexp"
	"strconv"
)

type Header struct {
	IP     net.IP
	Filter string
}

type T struct {
	Src net.TCPAddr
	Dst net.TCPAddr
	Raw []byte
}

type Data struct {
	H Header
	T []T
}

var (
	headre = [...]*regexp.Regexp{
		regexp.MustCompile(`interface: [\w\d]+ \(([\.:\w\d]+)\/.*\)`),
		regexp.MustCompile(`filter: (.*)`),
	}
	tre = regexp.MustCompile(`T ([\.:\w\d]+) -> ([\.:\w\d]+)`)
)

func parseAddr(s string) (addr net.TCPAddr, err error) {
	host, port, err := net.SplitHostPort(s)
	if err != nil {
		return
	}
	ip := net.ParseIP(host)
	if ip == nil {
		err = fmt.Errorf("invalid IP: %s", host)
		return
	}
	p, err := strconv.Atoi(port)
	if err != nil {
		return
	}
	addr = net.TCPAddr{IP: ip, Port: p}
	return
}

func parse(r io.Reader) (*Data, error) {
	type state uint8
	const (
		stHead state = iota
		stT
		stRaw
	)
	var (
		t   *T
		d   = &Data{T: make([]T, 0)}
		buf = bufio.NewReader(r)
		st  = stHead
	)
	for {
		b, err := buf.ReadBytes('\n')
		if err == io.EOF {
			return d, nil
		}
		if err != nil {
			return nil, err
		}
		switch st {
		case stRaw:
			if len(b) == 1 && b[0] == '\n' {
				st = stT
				continue
			}
			if b[len(b)-2] == '.' {
				b[len(b)-2] = '\r'
			}
			t.Raw = append(t.Raw, b...)
		case stT:
			if m := tre.FindStringSubmatch(string(b)); m != nil {
				d.T = append(d.T, T{})
				t = &d.T[len(d.T)-1]
				if t.Src, err = parseAddr(m[1]); err != nil {
					return nil, err
				}
				if t.Dst, err = parseAddr(m[2]); err != nil {
					return nil, err
				}
				st = stRaw
			}
		case stHead:
			if len(b) == 1 && b[0] == '\n' {
				st = stT
				continue
			}
			line := string(b)
			if m := headre[0].FindStringSubmatch(line); m != nil {
				if d.H.IP = net.ParseIP(m[1]); d.H.IP == nil {
					return nil, fmt.Errorf("invalid IP: %s", m[1])
				}
			} else if m := headre[1].FindStringSubmatch(line); m != nil {
				d.H.Filter = m[1]
			}
		}
	}
}

var r io.ReadCloser

func fatal(err error) {
	fmt.Fprintf(os.Stderr, "xmlrpcgen: %v\n", err)
	os.Exit(1)
}

func init() {
	file := flag.String("i", "", "input file")
	flag.Parse()
	if len(*file) == 0 {
		fatal(errors.New("missing value for -i flag"))
	}
	var err error
	if r, err = os.Open(*file); err != nil {
		fatal(err)
	}
}

func main() {
	d, err := parse(r)
	if err != nil {
		fatal(err)
	}
	fmt.Printf("%v\n", *d)
}
