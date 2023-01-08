package main

import (
	"crypto/tls"
	"encoding/binary"
	"fmt"
	quicconn "github.com/marten-seemann/quic-conn"
	"io"
	"time"
)

func main() {
	// run the client
	go func() {
		tlsConf := &tls.Config{InsecureSkipVerify: true,
			NextProtos: []string{"quic-echo-server"}}
		conn, err := quicconn.Dial("localhost:4242", tlsConf)
		if err != nil {
			panic(err)
		}

		message := []byte("Ping from client")
		prefix := make([]byte, 4)
		binary.BigEndian.PutUint32(prefix, uint32(len(message)))

		// write the prefix and the data to the stream (checking errors)
		_, err = conn.Write(prefix)
		_, err = conn.Write(message)

		//conn.Write(message)
		fmt.Println("message len: ", prefix)
		fmt.Printf("message:\t %v \n", message)
		// listen for reply
		//answer, err := bufio.NewReader(conn).ReadBytes('\n')
		msg_len_bytes := make([]byte, 4)
		_, err = io.ReadFull(conn, msg_len_bytes)
		msg_len := binary.BigEndian.Uint32(msg_len_bytes)
		msg := make([]byte, msg_len)
		_, err = io.ReadFull(conn, msg)
		if err != nil {
			panic(err)
		}
		fmt.Print("Message from server: " + string(msg))
	}()
	time.Sleep(time.Hour)
}
