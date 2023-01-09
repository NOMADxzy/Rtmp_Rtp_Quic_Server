package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/binary"
	"encoding/gob"
	"encoding/pem"
	"errors"
	"fmt"
	"github.com/lucas-clemente/quic-go"
	"reflect"

	//quicconn "github.com/marten-seemann/quic-conn"

	"math/big"
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
			c.infoStream, err = c.session.AcceptStream(context.Background())
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
func (c *conn) Write(b []byte, write_data bool) (int, error) {
	if write_data {
		return c.dataStream.Write(b)
	} else {
		return c.infoStream.Write(b)
	}

}

func (c *conn) SendRtp(pkt RTPPacket) (int, error) {
	_, err := c.dataStream.Write(pkt.buffer)
	if err != nil {
		panic(err)
	}
	return c.dataStream.Write(pkt.ekt)
}

func main() {
	// start the server
	go func() {
		tlsConf, err := generateTLSConfig()
		if err != nil {
			panic(err)
		}

		ln, err := quic.ListenAddr("localhost:4242", tlsConf, nil)

		if err != nil {
			panic(err)
		}

		fmt.Println("Waiting for incoming connection")
		protoconn, err := ln.Accept(context.Background())

		conn, _ := newConn(protoconn, true)

		if err != nil {
			panic(err)
		}
		fmt.Println("Established connection")

		seq_b := make([]byte, 4)

		_, err = conn.Read(seq_b, false)
		seq := binary.BigEndian.Uint32(seq_b)
		//msg := make([]byte, msg_len+10)
		//
		//_, err = conn.Read(msg)
		//if err != nil {
		//	panic(err)
		//}
		fmt.Println("seq: ", seq)

		//发送rtp数据包给客户

		rtp := []byte("Rtp data...")
		payload := make([]byte, 10)
		for i := range payload {
			payload[i] = byte(i)
		}
		pkt := NewRTPPacket(payload, int8(0), uint16(1), uint32(2), uint32(3))

		len_b := make([]byte, 4)
		binary.BigEndian.PutUint32(len_b, uint32(len(rtp)))
		_, err = conn.Write(len_b, false)
		if err != nil {
			panic(err)
		}
		_, err = conn.Write(rtp, true)
		if err != nil {
			panic(err)
		}
	}()

	time.Sleep(time.Hour)
}

func generateTLSConfig() (*tls.Config, error) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, err
	}
	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		NotBefore:    time.Now(),
		NotAfter:     time.Now().Add(time.Hour),
		KeyUsage:     x509.KeyUsageCertSign | x509.KeyUsageDigitalSignature,
	}
	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &key.PublicKey, key)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}
	keyPEM := pem.EncodeToMemory(&pem.Block{
		Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key),
	})
	b := pem.Block{Type: "CERTIFICATE", Bytes: certDER}
	certPEM := pem.EncodeToMemory(&b)

	tlsCert, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		return nil, err
	}

	return &tls.Config{
		Certificates: []tls.Certificate{tlsCert},
		NextProtos:   []string{"quic-echo-server"},
	}, nil
}
func Encoder(inter interface{}) bytes.Buffer {
	var buf bytes.Buffer
	// writer
	encoder := gob.NewEncoder(&buf)
	err := encoder.Encode(inter)
	if err != nil {
		fmt.Println(err)
	}
	return buf
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
