package fakerpc

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"net"
	"regexp"
	"strconv"
)

var (
	headre = [...]*regexp.Regexp{
		regexp.MustCompile(`interface: [\w\d]+ \(([\.:\w\d]+)\/([\.:\w\d]+)\)`),
		regexp.MustCompile(`filter: (.*)`),
	}
	tre = regexp.MustCompile(`T ([\.:\w\d]+) -> ([\.:\w\d]+)`)
)

func parseAddr(s string) (addr *net.TCPAddr, err error) {
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
	addr = &net.TCPAddr{IP: ip, Port: p}
	return
}

// NgrepUnmarshal parses the ngrep output read from r and stores the result
// in the l.
func NgrepUnmarshal(r io.Reader, l *Log) error {
	type state uint8
	const (
		stHead state = iota
		stT
		stRaw
	)
	var (
		t   *Transmission
		buf = bufio.NewReader(r)
		st  = stHead
	)
	for {
		b, err := buf.ReadBytes('\n')
		if len(b) == 0 {
			if err == io.EOF {
				return nil
			}
			if err != nil {
				return err
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
					return err
				}
				if t.Dst, err = parseAddr(m[2]); err != nil {
					return err
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
					return errors.New("ill-formed IP " + m[1])
				}
				mask := net.ParseIP(m[2])
				if mask == nil {
					return errors.New("ill-formed IP mask " + m[2])
				}
				l.Network.Mask = iptomask(mask)
			} else if m := headre[1].FindStringSubmatch(line); m != nil {
				l.Filter = m[1]
			}
		}
	}
}

// NgrepMarshal writes to w the l encoded as a ngrep output.
func NgrepMarshal(w io.Writer, l *Log) (err error) {
	_, err = fmt.Fprintf(w, "interface: dunno0 (%s)\n", l.Net())
	if err != nil {
		return
	}
	if _, err = fmt.Fprintf(w, "filter: %s\n", l.Filter); err != nil {
		return
	}
	for i := range l.T {
		_, err = fmt.Fprintf(w, "\nT %s -> %s [AP]\n", l.T[i].Src.String(), l.T[i].Dst.String())
		if err != nil {
			return
		}
		var (
			b []byte
			r = bufio.NewReader(bytes.NewBuffer(l.T[i].Raw))
		)
		for {
			b, err = r.ReadBytes('\n')
			if len(b) == 0 {
				if err == io.EOF {
					err = nil
					break
				}
				if err != nil {
					return
				}
			}
			if len(b) > 1 && b[len(b)-2] == '\r' {
				b[len(b)-2] = '.'
			}
			if b[len(b)-1] != '\n' {
				b = append(b, '\n')
			}
			if _, err = fmt.Fprintf(w, "%s", b); err != nil {
				return
			}
		}
	}
	return
}
