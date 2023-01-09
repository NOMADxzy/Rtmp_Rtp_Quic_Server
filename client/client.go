package main

import (
	"crypto/tls"
	"fmt"
	"github.com/lucas-clemente/quic-go"
	//quicconn "github.com/marten-seemann/quic-conn"
	//"io"
	"time"
)

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

		// 根据序列号请求
		_, err = conn.WriteSeq(uint16(seq))

		//读rtp数据
		pkt := RTPPacket{}
		_, err = conn.ReadRtp(&pkt)
		if err != nil {
			fmt.Println(err)
		}
		fmt.Printf("buf:\t %v \n", pkt.buffer)
		fmt.Printf("ekt:\t %v \n", pkt.ekt)
		fmt.Printf("Seq:\t %v \n", pkt.GetSeq())
		fmt.Printf("SSRC:\t %v \n", pkt.GetSSRC())
		fmt.Printf("ExtLen:\t %v \n", pkt.GetHdrExtLen())
		fmt.Printf("PTtype:\t %v \n", pkt.GetPT())
		fmt.Printf("Payload:\t %v \n", pkt.GetPayload())
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
