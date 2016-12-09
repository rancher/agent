package tls

import (
	"bufio"
	"crypto/tls"
	"net"
	"time"
)

type Conn struct {
	net.Conn
	bufReader *bufio.Reader
	err       error
}

func (c *Conn) Read(b []byte) (int, error) {
	if c.err != nil {
		return 0, c.err
	}
	return c.bufReader.Read(b)
}

type SplitListener struct {
	net.Listener
	Config *tls.Config
}

func (l *SplitListener) Accept() (net.Conn, error) {
	c, err := l.Listener.Accept()
	if err != nil {
		return nil, err
	}

	newConn := &Conn{
		Conn:      c,
		bufReader: bufio.NewReader(c),
	}

	c.SetDeadline(time.Now().Add(1 * time.Second))
	b, err := newConn.bufReader.Peek(1)
	if err != nil {
		newConn.err = err
		return newConn, nil
	}
	c.SetDeadline(time.Time{})

	if b[0] < 32 || b[0] >= 127 {
		return tls.Server(newConn, l.Config), nil
	}

	return newConn, nil
}
