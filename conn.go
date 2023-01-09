package quicconn

import (
	"context"
	"net"
	"time"

	quic "github.com/lucas-clemente/quic-go"
)

type conn struct {
	session    quic.Session
	infoStream quic.Stream
	dataStream quic.Stream
}

func newConn(sess quic.Session) (*conn, error) {
	istream, err := sess.OpenStream()
	dstream, err := sess.OpenStream()
	if err != nil {
		return nil, err
	}
	return &conn{
		session:    sess,
		infoStream: istream,
		dataStream: dstream,
	}, nil
}

func (c *conn) DataStream() quic.Stream {
	return c.dataStream
}
func (c *conn) Read(b []byte) (int, error) {

	if c.dataStream == nil {
		var err error
		c.dataStream, err = c.session.AcceptStream(context.Background())
		// TODO: check stream id
		if err != nil {
			return 0, err
		}
		// quic.Stream.Close() closes the stream for writing
		//err = c.dataStream.Close()
		//if err != nil {
		//	return 0, err
		//}
	}

	return c.dataStream.Read(b)
	//return io.ReadFull(c.dataStream,b)
}

func (c *conn) Write(b []byte) (int, error) {
	return c.dataStream.Write(b)
}

// LocalAddr returns the local network address.
// needed to fulfill the net.Conn interface
func (c *conn) LocalAddr() net.Addr {
	return c.session.LocalAddr()
}

// RemoteAddr returns the remote network address.
func (c *conn) RemoteAddr() net.Addr {
	return c.session.RemoteAddr()
}

func (c *conn) Close() error {
	return c.session.Close()
}

func (c *conn) SetDeadline(t time.Time) error {
	return nil
}

func (c *conn) SetReadDeadline(t time.Time) error {
	return nil
}

func (c *conn) SetWriteDeadline(t time.Time) error {
	return nil
}

var _ net.Conn = &conn{}
