package black

import (
	"bytes"
	"compress/gzip"
	"errors"
	"io"
	"log"
	"net"
	"time"
)

var errNotImplemtented = errors.New("bx: not implemented")

// RunCloser TODO
type RunCloser interface {
	Run() error
	Close() error
}

type lis struct {
	net.Listener
	s *Session
}

func (l *lis) Accept() (c net.Conn, err error) {
	if c, err = l.Listener.Accept(); err != nil {
		return
	}
	(*l.s) = append((*l.s), make(Connection, 0, 2))
	c = &conn{Conn: c, c: &(*l.s)[len(*l.s)-1]}
	return
}

type conn struct {
	net.Conn
	c *Connection
}

func (c *conn) Read(p []byte) (n int, err error) {
	n, err = c.Conn.Read(p)
	c.c.Append(p, Response)
	return
}

func (c *conn) Write(p []byte) (n int, err error) {
	n, err = c.Conn.Write(p)
	c.c.Append(p, Request)
	return
}

// GzipBuffer TODO
type GzipBuffer struct {
	bytes.Buffer
}

func (b *GzipBuffer) MarshalBinary() ([]byte, error) {
	var buf bytes.Buffer
	w := gzip.NewWriter(&buf)
	defer w.Close()
	if _, err := io.Copy(w, b); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (b *GzipBuffer) UnmarshalBinary(data []byte) error {
	var buf = bytes.NewBuffer(data)
	r, err := gzip.NewReader(buf)
	if err != nil {
		return err
	}
	defer r.Close()
	_, err = io.Copy(b, r)
	return err
}

type MessageType uint8

const (
	Request MessageType = iota + 1
	Response
)

// Message TODO
type Message struct {
	Start time.Time
	End   time.Time
	Raw   GzipBuffer
	Type  MessageType
}

// Connection TODO
type Connection []Message

// Append TODO
func (c *Connection) Append(data []byte, typ MessageType) {
	log.Printf("[DEBUG] appending %d bytes . . .", len(data))
	if len(*c) != 0 && (*c)[len(*c)-1].Type != typ {
		(*c)[len(*c)-1].End = time.Now()
	}
	if len(*c) == 0 || (*c)[len(*c)-1].Type != typ {
		*c = append(*c, Message{Start: time.Now(), Type: typ})
	}
	(*c)[len(*c)-1].Raw.Write(data)
	log.Printf("[DEBUG] buffer is %d bytes . . .", (*c)[len(*c)-1].Raw.Len())
}

// Session TODO
type Session []Connection

// NewSession TODO
func NewSession() Session {
	return make(Session, 0, 1)
}

// Wiretap TODO
func (s *Session) Wiretap(l net.Listener) net.Listener {
	return &lis{Listener: l, s: s}
}

// Header TODO
type Header struct {
	Version uint8
	Scheme  string
	Target  string
}

func NewHeader(scheme, target string) Header {
	return Header{1, scheme, target}
}

// SessionFile
type SessionFile struct {
	*Header
	*Session
}
