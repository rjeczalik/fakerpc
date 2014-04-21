package fakerpc

import (
	"bytes"
	"net"
	"reflect"
	"testing"
)

const ngrep = `interface: tun0 (192.168.14.0/255.255.255.0)
filter: (ip or ip6) and ( host 192.168.16.50 and port 80 )

T 192.168.14.108:46793 -> 192.168.16.50:80 [AP]
REQ.
UE.
ST

T 192.168.16.50:80 -> 192.168.14.108:46793 [AP]
RES.
PONSE

T 192.168.14.108:46794 -> 192.168.16.50:80 [AP]
REQUEST.

T 192.168.14.108:46794 -> 192.168.16.50:80 [AP]
MORE.

T 192.168.16.50:80 -> 192.168.14.108:46794 [AP]
RES.
PON.
SE

T 192.168.14.108:46794 -> 192.168.16.50:80 [AP]
KTHXBAI

T 192.168.14.108:46795 -> 192.168.16.50:80 [AP]
LAST.
ONE

T 192.168.16.50:80 -> 192.168.14.108:46795 [AP]
KK

T 192.168.14.108:46795 -> 192.168.16.50:80 [AP]
I.
LIED`

var lexp = &Log{
	Network: net.IPNet{
		IP:   net.IPv4(192, 168, 14, 0),
		Mask: net.IPv4Mask(255, 255, 255, 0),
	},
	Filter: "(ip or ip6) and ( host 192.168.16.50 and port 80 )",
	T: []Transmission{{
		Src: net.TCPAddr{
			IP:   net.IPv4(192, 168, 14, 108),
			Port: 46793,
		},
		Dst: net.TCPAddr{
			IP:   net.IPv4(192, 168, 16, 50),
			Port: 80,
		},
		Raw: []byte("REQ\r\nUE\r\nST\n"),
	}, {
		Src: net.TCPAddr{
			IP:   net.IPv4(192, 168, 16, 50),
			Port: 80,
		},
		Dst: net.TCPAddr{
			IP:   net.IPv4(192, 168, 14, 108),
			Port: 46793,
		},
		Raw: []byte("RES\r\nPONSE\n"),
	}, {
		Src: net.TCPAddr{
			IP:   net.IPv4(192, 168, 14, 108),
			Port: 46794,
		},
		Dst: net.TCPAddr{
			IP:   net.IPv4(192, 168, 16, 50),
			Port: 80,
		},
		Raw: []byte("REQUEST\r\n"),
	}, {
		Src: net.TCPAddr{
			IP:   net.IPv4(192, 168, 14, 108),
			Port: 46794,
		},
		Dst: net.TCPAddr{
			IP:   net.IPv4(192, 168, 16, 50),
			Port: 80,
		},
		Raw: []byte("MORE\r\n"),
	}, {
		Src: net.TCPAddr{
			IP:   net.IPv4(192, 168, 16, 50),
			Port: 80,
		},
		Dst: net.TCPAddr{
			IP:   net.IPv4(192, 168, 14, 108),
			Port: 46794,
		},
		Raw: []byte("RES\r\nPON\r\nSE\n"),
	}, {
		Src: net.TCPAddr{
			IP:   net.IPv4(192, 168, 14, 108),
			Port: 46794,
		},
		Dst: net.TCPAddr{
			IP:   net.IPv4(192, 168, 16, 50),
			Port: 80,
		},
		Raw: []byte("KTHXBAI\n"),
	}, {
		Src: net.TCPAddr{
			IP:   net.IPv4(192, 168, 14, 108),
			Port: 46795,
		},
		Dst: net.TCPAddr{
			IP:   net.IPv4(192, 168, 16, 50),
			Port: 80,
		},
		Raw: []byte("LAST\r\nONE\n"),
	}, {
		Src: net.TCPAddr{
			IP:   net.IPv4(192, 168, 16, 50),
			Port: 80,
		},
		Dst: net.TCPAddr{
			IP:   net.IPv4(192, 168, 14, 108),
			Port: 46795,
		},
		Raw: []byte("KK\n"),
	}, {
		Src: net.TCPAddr{
			IP:   net.IPv4(192, 168, 14, 108),
			Port: 46795,
		},
		Dst: net.TCPAddr{
			IP:   net.IPv4(192, 168, 16, 50),
			Port: 80,
		},
		Raw: []byte("I\r\nLIED"),
	}},
}

func TestParseNgrep(t *testing.T) {
	l, err := ParseNgrep(bytes.NewBufferString(ngrep))
	if err != nil {
		t.Fatalf("expected err=nil; was %q", err)
	}
	if !l.Network.IP.Equal(lexp.Network.IP) {
		t.Errorf("expected l.Network.IP=%q; was %q", lexp.Network.IP,
			l.Network.IP)
	}
	if l.Network.Mask.String() != lexp.Network.Mask.String() {
		t.Errorf("expected l.Network.Mask=%q; was %q", lexp.Network.Mask,
			l.Network.Mask)
	}
	if len(l.T) != len(lexp.T) {
		t.Fatalf("expected len(l.T)=%d; was %d", len(lexp.T), len(l.T))
	}
	for i := range lexp.T {
		if !reflect.DeepEqual(l.T[i].Src, lexp.T[i].Src) {
			t.Errorf("expected l.T[%d].Src=%v; was %v", i, lexp.T[i].Src, l.T[i].Src)
		}
		if !reflect.DeepEqual(l.T[i].Dst, lexp.T[i].Dst) {
			t.Errorf("expected l.T[%d].Dst=%v; was %v", i, lexp.T[i].Dst, l.T[i].Dst)
		}
		if !bytes.Equal(l.T[i].Raw, lexp.T[i].Raw) {
			t.Errorf("expected l.T[%d].Raw=%q; was %q", i, lexp.T[i].Raw, l.T[i].Raw)
		}
	}
}
