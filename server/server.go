package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/binary"
	"encoding/pem"
	"fmt"
	"io"
	"math/big"
	"time"

	quicconn "github.com/marten-seemann/quic-conn"
)

func main() {
	// start the server
	go func() {
		tlsConf, err := generateTLSConfig()
		if err != nil {
			panic(err)
		}

		ln, err := quicconn.Listen("udp", ":4242", tlsConf)

		if err != nil {
			panic(err)
		}

		fmt.Println("Waiting for incoming connection")
		conn, err := ln.Accept()
		if err != nil {
			panic(err)
		}
		fmt.Println("Established connection")

		msg_len_bytes := make([]byte, 4)
		_, err = io.ReadFull(conn, msg_len_bytes)
		msg_len := binary.BigEndian.Uint32(msg_len_bytes)
		msg := make([]byte, msg_len)

		//_, err = io.ReadFull(conn, msg)
		_, err = conn.Read(msg)
		if err != nil {
			panic(err)
		}
		fmt.Println("msg_len: ", msg_len)
		fmt.Println("Message from client: " + string(msg))
		//message, err := bufio.NewReader(conn).ReadBytes('\n')
		//if err != nil {
		//	panic(err)
		//}
		//fmt.Print("Message from client: ", string(message))
		// echo back
		//newmessage := strings.ToUpper(message)

		//发送rtp数据包给客户
		_, err = conn.Write(msg_len_bytes)
		if err != nil {
			panic(err)
		}
		_, err = conn.Write(msg)
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
