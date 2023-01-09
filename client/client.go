package main

import (
	"bytes"
	"crypto/tls"
	"encoding/binary"
	"encoding/gob"
	"errors"
	"fmt"
	"reflect"

	"context"
	"github.com/lucas-clemente/quic-go"
	//quicconn "github.com/marten-seemann/quic-conn"
	//"io"
	"time"
)

type conn struct {
	session    quic.Session
	infoStream quic.Stream
	dataStream quic.Stream
}

func newConn(sess quic.Session, is_server bool) (*conn, error) {
	if is_server {
		dstream, err := sess.OpenStream()
		if err != nil {
			return nil, err
		}
		return &conn{
			session:    sess,
			dataStream: dstream,
		}, nil
	} else {
		istream, err := sess.OpenStream()
		if err != nil {
			return nil, err
		}
		return &conn{
			session:    sess,
			infoStream: istream,
		}, nil
	}
}

//func (c *conn) DataStream() quic.Stream {
//	return c.dataStream
//}
func (c *conn) Read(b []byte, isdata bool) (int, error) {
	var stream quic.Stream
	if isdata {
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
		stream = c.dataStream
	} else {
		if c.infoStream == nil {
			var err error
			c.dataStream, err = c.session.AcceptStream(context.Background())
			// TODO: check stream id
			if err != nil {
				return 0, err
			}
		}
		stream = c.infoStream
	}
	return stream.Read(b)
	//return io.ReadFull(c.dataStream,b)
}
func (c *conn) Write(b []byte) (int, error) {
	return c.infoStream.Write(b)
}
func main() {
	// run the client
	go func() {
		tlsConf := &tls.Config{InsecureSkipVerify: true,
			NextProtos: []string{"quic-echo-server"}}
		protoconn, err := quic.DialAddr("localhost:4242", tlsConf, nil)
		if err != nil {
			panic(err)
		}
		conn, _ := newConn(protoconn, false)

		seq := 6261
		seq_b := make([]byte, 4)
		binary.BigEndian.PutUint32(seq_b, uint32(seq))

		// write the prefix and the data to the stream (checking errors)
		_, err = conn.Write(seq_b)
		rtp_len_b := make([]byte, 4)
		_, err = conn.Read(rtp_len_b, false)
		if err != nil {
			panic(err)
		}
		rtp_len := binary.BigEndian.Uint32(rtp_len_b)

		rtp_b := make([]byte, rtp_len)
		_, err = conn.Read(rtp_b, true)
		if err != nil {
			panic(err)
		}
		fmt.Println("rtp length: ", rtp_len)

		if err != nil {
			fmt.Println(err)
		}
		fmt.Print("rtp: " + string(rtp_b))
		//_, err = conn.Write(message)

		//fmt.Println("message len: ", prefix)
		//fmt.Printf("message:\t %v \n", message)
		//msg_len_bytes := make([]byte, 4)
		//_, err = io.ReadFull(conn, msg_len_bytes)
		//msg_len := binary.BigEndian.Uint32(msg_len_bytes)
		//msg := make([]byte, msg_len)
		//_, err = io.ReadFull(conn, msg)
		//if err != nil {
		//	panic(err)
		//}
		//fmt.Print("Message from server: " + string(msg))
	}()
	time.Sleep(time.Hour)
}
func Decoder(buf bytes.Buffer, inter interface{}) error {
	rt := reflect.TypeOf(inter)
	if rt.Kind() != reflect.Ptr {
		return errors.New("inter must be ptr")
	}
	// reader
	decoder := gob.NewDecoder(&buf)
	decoder.Decode(inter)
	return nil
}
