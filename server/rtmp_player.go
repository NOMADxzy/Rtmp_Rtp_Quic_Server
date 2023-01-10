package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"net/rtp"
	"os"
	"time"

	"github.com/zhangpeihao/goflv"
	rtmp "github.com/zhangpeihao/gortmp"
	"github.com/zhangpeihao/log"
)

const (
	programName = "RtmpPlayer"
	version     = "0.0.1"
)

var (
	url        *string = flag.String("URL", "rtmp://127.0.0.1:1935/flv_test", "The rtmp url to connect.")
	streamName *string = flag.String("Stream", "test", "Stream name to play.")
	dumpFlv    *string = flag.String("DumpFLV", "./recv.flv", "Dump FLV into file.")
)

type TestOutboundConnHandler struct {
}

var obConn rtmp.OutboundConn
var createStreamChan chan rtmp.OutboundStream
var videoDataSize int64
var audioDataSize int64
var flvFile *flv.File
var status uint

func (handler *TestOutboundConnHandler) OnStatus(conn rtmp.OutboundConn) {
	var err error
	status, err = conn.Status()
	fmt.Printf("@@@@@@@@@@@@@status: %d, err: %v\n", status, err)
}

func (handler *TestOutboundConnHandler) OnClosed(conn rtmp.Conn) {
	fmt.Printf("@@@@@@@@@@@@@Closed\n")
}

func (handler *TestOutboundConnHandler) OnReceived(conn rtmp.Conn, message *rtmp.Message) {
	//fmt.Println("recv message size: ", message.Size)
	if len(message.Buf.Bytes()) < 1000 {
		new_pkt := NewRTPPacket(message.Buf.Bytes(), int8(9), CUR_SEQ, uint32(2), uint32(3))
		rtp_queue.Enqueue(new_pkt, CUR_SEQ)
		CUR_SEQ += uint16(1)
		fmt.Println("当前rtp队列长度：", rtp_queue.queue.Len(), " 队列数据量：", rtp_queue.bytesInQueue)
	}

	switch message.Type {
	case rtmp.VIDEO_TYPE:
		rp := rsLocal.NewDataPacket(uint32(0))
		rp.SetPayload(message.Buf.Bytes())
		rsLocal.WriteData(rp)
		rp.FreePacket()
		if flvFile != nil {
			flvFile.WriteVideoTag(message.Buf.Bytes(), message.AbsoluteTimestamp)
		}
		videoDataSize += int64(message.Buf.Len())
	case rtmp.AUDIO_TYPE:
		if flvFile != nil {
			flvFile.WriteAudioTag(message.Buf.Bytes(), message.AbsoluteTimestamp)
		}
		audioDataSize += int64(message.Buf.Len())
	}
}

func (handler *TestOutboundConnHandler) OnReceivedRtmpCommand(conn rtmp.Conn, command *rtmp.Command) {
	fmt.Printf("ReceviedCommand: %+v\n", command)
}

func (handler *TestOutboundConnHandler) OnStreamCreated(conn rtmp.OutboundConn, stream rtmp.OutboundStream) {
	fmt.Printf("Stream created: %d\n", stream.ID())
	createStreamChan <- stream
}

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "%s version[%s]\r\nUsage: %s [OPTIONS]\r\n", programName, version, os.Args[0])
		flag.PrintDefaults()
	}
	flag.Parse()

	fmt.Printf("rtmp:%s stream:%s flv:%s\r\n", *url, *streamName, *dumpFlv)
	l := log.NewLogger(".", "player", nil, 60, 3600*24, true)
	rtmp.InitLogger(l)
	initialize()
	defer l.Close()

	//rtpsession初始化
	tpLocal, _ := rtp.NewTransportUDP(local, localPort, localZone)
	rsLocal = rtp.NewSession(tpLocal, tpLocal)
	rsLocal.AddRemote(&rtp.Address{remote.IP, remotePort, remotePort + 1, remoteZone})
	strLocalIdx, _ := rsLocal.NewSsrcStreamOut(&rtp.Address{local.IP, localPort, localPort + 1, localZone}, 1020304, CUR_SEQ)
	rsLocal.SsrcStreamOutForIndex(strLocalIdx).SetPayloadType(9)
	rsLocal.StartSession()
	defer rsLocal.CloseSession()

	// Create flv file
	if len(*dumpFlv) > 0 {
		var err error
		flvFile, err = flv.CreateFile(*dumpFlv)
		if err != nil {
			fmt.Println("Create FLV dump file error:", err)
			return
		}
	}
	defer func() {
		if flvFile != nil {
			flvFile.Close()
		}
	}()

	createStreamChan = make(chan rtmp.OutboundStream)
	testHandler := &TestOutboundConnHandler{}
	fmt.Println("to dial")
	var err error
	obConn, err = rtmp.Dial(*url, testHandler, 100)
	/*
		conn := TryHandshakeByVLC()
		obConn, err = rtmp.NewOutbounConn(conn, *url, testHandler, 100)
	*/
	if err != nil {
		fmt.Println("Dial error", err)
		os.Exit(-1)
	}

	defer obConn.Close()
	fmt.Printf("obConn: %+v\n", obConn)
	fmt.Printf("obConn.URL(): %s\n", obConn.URL())
	fmt.Println("to connect")
	//	err = obConn.Connect("33abf6e996f80e888b33ef0ea3a32bfd", "131228035", "161114738", "play", "", "", "1368083579")
	err = obConn.Connect()
	if err != nil {
		fmt.Printf("Connect error: %s", err.Error())
		os.Exit(-1)
	}

	go func() {
		conn := initialQUIC()
		var seq uint16
		for {

			_, err = conn.ReadSeq(&seq)
			if err != nil {
				panic(err)
			}
			//msg := make([]byte, msg_len+10)
			//
			//_, err = conn.Read(msg)
			fmt.Println("seq: ", seq)

			//发送rtp数据包给客户
			pkt := rtp_queue.GetPkt(seq)
			if pkt != nil {
				_, err = conn.SendRtp(*pkt)
				if err != nil {
					panic(err)
				}
			}
		}
	}()

	for {
		select {
		case stream := <-createStreamChan:
			// Play
			err = stream.Play(*streamName, nil, nil, nil)
			if err != nil {
				fmt.Printf("Play error: %s", err.Error())
				os.Exit(-1)
			}
			// Set Buffer Length

		case <-time.After(1 * time.Second):
			fmt.Printf("Audio size: %d bytes; Vedio size: %d bytes\n", audioDataSize, videoDataSize)
		}
	}
}

////////////////////////////////////////////
func CheckC1(c1 []byte, offset1 bool) (uint32, error) {
	var clientDigestOffset uint32
	if offset1 {
		clientDigestOffset = rtmp.CalcDigestPos(c1, 8, 728, 12)
	} else {
		clientDigestOffset = rtmp.CalcDigestPos(c1, 772, 728, 776)
	}
	// Create temp buffer
	tmpBuf := new(bytes.Buffer)
	tmpBuf.Write(c1[:clientDigestOffset])
	tmpBuf.Write(c1[clientDigestOffset+rtmp.SHA256_DIGEST_LENGTH:])
	// Generate the hash
	tempHash, err := rtmp.HMACsha256(tmpBuf.Bytes(), rtmp.GENUINE_FP_KEY[:30])
	if err != nil {
		return 0, errors.New(fmt.Sprintf("HMACsha256 err: %s\n", err.Error()))
	}
	expect := c1[clientDigestOffset : clientDigestOffset+rtmp.SHA256_DIGEST_LENGTH]
	if bytes.Compare(expect, tempHash) != 0 {
		return 0, errors.New(fmt.Sprintf("C1\nExpect % 2x\nGot    % 2x\n",
			expect,
			tempHash))
	}
	return clientDigestOffset, nil
}
