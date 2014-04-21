package fakerpc

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"net"
	"regexp"
	"strconv"
)

// Transmission TODO(rjeczalik): document
type Transmission struct {
	Src net.TCPAddr
	Dst net.TCPAddr
	Raw []byte
}

// Log TODO(rjeczalik): document
type Log struct {
	Network net.IPNet
	Filter  string
	T       []Transmission
}

var (
	headre = [...]*regexp.Regexp{
		regexp.MustCompile(`interface: [\w\d]+ \(([\.:\w\d]+)\/([\.:\w\d]+)\)`),
		regexp.MustCompile(`filter: (.*)`),
	}
	tre = regexp.MustCompile(`T ([\.:\w\d]+) -> ([\.:\w\d]+)`)
)

// ParseNgrep TODO(rjeczalik): document
func ParseNgrep(r io.Reader) (*Log, error) {
	type state uint8
	const (
		stHead state = iota
		stT
		stRaw
	)
	var (
		t   *Transmission
		l   = &Log{T: make([]Transmission, 0)}
		buf = bufio.NewReader(r)
		st  = stHead
	)
	for {
		b, err := buf.ReadBytes('\n')
		if len(b) == 0 {
			if err == io.EOF {
				return l, nil
			}
			if err != nil {
				return nil, err
			}
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
				l.T = append(l.T, Transmission{})
				t = &l.T[len(l.T)-1]
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
				if l.Network.IP = net.ParseIP(m[1]); l.Network.IP == nil {
					return nil, errors.New("ill-formed IP " + m[1])
				}
				mask := net.ParseIP(m[2])
				if mask == nil {
					return nil, errors.New("ill-formed IP mask " + m[2])
				}
				l.Network.Mask = net.IPv4Mask(mask[12], mask[13], mask[14], mask[15])
			} else if m := headre[1].FindStringSubmatch(line); m != nil {
				l.Filter = m[1]
			}
		}
	}
}

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
