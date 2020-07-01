package protolistener

import (
	"bufio"
	"net"

	log "github.com/fangdingjun/go-log/v5"
	proxyproto "github.com/pires/go-proxyproto"
)

type protoListener struct {
	net.Listener
}

type protoConn struct {
	net.Conn
	headerDone bool
	r          *bufio.Reader
	proxy      *proxyproto.Header
	err        error
}

// New create a wrapped listener
func New(l net.Listener) net.Listener {
	return &protoListener{l}
}

func (l *protoListener) Accept() (net.Conn, error) {
	c, err := l.Listener.Accept()
	if err != nil {
		return nil, err
	}
	return &protoConn{Conn: c}, err
}

func (c *protoConn) readHeader() error {
	var err error
	c.r = bufio.NewReader(c.Conn)
	c.proxy, err = proxyproto.Read(c.r)
	if err != nil && err != proxyproto.ErrNoProxyProtocol {
		c.err = err
		return err
	}
	return nil
}

func (c *protoConn) Read(buf []byte) (int, error) {
	if c.err != nil {
		return 0, c.err
	}
	if !c.headerDone {
		if err := c.readHeader(); err != nil {
			c.headerDone = true
			return 0, err
		}
		c.headerDone = true
		return c.r.Read(buf)
	}
	return c.r.Read(buf)
}

func (c *protoConn) RemoteAddr() net.Addr {
	if !c.headerDone {
		if err := c.readHeader(); err != nil {
			log.Errorln(err)
		}
		c.headerDone = true
	}
	if c.proxy == nil {
		return c.Conn.RemoteAddr()
	}
	return &net.TCPAddr{
		IP:   c.proxy.SourceAddress,
		Port: int(c.proxy.SourcePort)}
}
