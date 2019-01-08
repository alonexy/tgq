package main

// Start Commond eg: ./client 1 1000 localhost:3101
// first parameter：beginning userId
// second parameter: amount of clients
// third parameter: comet server ip

import (
	"bufio"
	"encoding/binary"
	"encoding/json"
	"net"
	"os"
	"sync/atomic"
	"time"

	"log"
)

const (
	opHeartbeat      = int32(2)
	opHeartbeatReply = int32(3)
	opAuth           = int32(7)
	opAuthReply      = int32(8)
)

const (
	rawHeaderLen = uint16(16)

	heart        = 30 * time.Second
)

// Proto proto.
type Proto struct {
	PackLen   int32  // package length
	HeaderLen int16  // header length
	Ver       int16  // protocol version
	Operation int32  // operation for request
	Seq       int32  // sequence number chosen by client
	Body      []byte // body
}

// AuthToken auth token.
type AuthToken struct {
	Mid      int64   `json:"mid"`
	Key      string  `json:"key"`
	RoomID   string  `json:"room_id"`
	Platform string  `json:"platform"`
	Accepts  []int32 `json:"accepts"`
}

var (
	countDown  int64
	aliveCount int64
)

func main() {
	startClient(1)
}

func startClient(key int64) {
	//time.Sleep(time.Duration(rand.Intn(120)) * time.Second)
	atomic.AddInt64(&aliveCount, 1)
	quit := make(chan bool, 1)
	defer func() {
		close(quit)
		atomic.AddInt64(&aliveCount, -1)
	}()
	// connnect to server
	conn, err := net.Dial("tcp", "127.0.0.1:3101")
	if err != nil {
		log.Printf("net.Dial(%s) error(%v)", os.Args[3], err)
		return
	}
	seq := int32(0)
	wr := bufio.NewWriter(conn)
	rd := bufio.NewReader(conn)
	authToken := &AuthToken{
		key,
		"",
		"live://1000",
		"ios",
		[]int32{1000, 1001, 1002},
	}
	proto := new(Proto)
	proto.Ver = 1
	proto.Operation = opAuth
	proto.Seq = seq
	proto.Body, _ = json.Marshal(authToken)
	if err = tcpWriteProto(wr, proto); err != nil {
		log.Printf("tcpWriteProto() error(%v)", err)
		return
	}
	if err = tcpReadProto(rd, proto); err != nil {
		log.Printf("tcpReadProto() error(%v)", err.Error())
		return
	}
	log.Printf("key:%d auth ok, proto: %v", key, proto)
	seq++
	// writer
	go func() {
		hbProto := new(Proto)
		for {
			// heartbeat
			hbProto.Operation = opHeartbeat
			hbProto.Seq = seq
			hbProto.Body = nil
			if err = tcpWriteProto(wr, hbProto); err != nil {
				log.Printf("key:%d tcpWriteProto() error(%v)", key, err)
				return
			}
			log.Printf("key:%d Write heartbeat", key)
			time.Sleep(heart)
			seq++
			select {
			case <-quit:
				return
			default:
			}
		}
	}()
	// reader
	for {
		if err = tcpReadProto(rd, proto); err != nil {
			log.Printf("key:%d tcpReadProto() error(%v)", key, err)
			quit <- true
			return
		}
		if proto.Operation == opAuthReply {
			log.Printf("key:%d auth success", key)
			//hbProto := new(Proto)
			//hbProto.Operation = opHeartbeat
			//hbProto.Seq = seq
			//hbProto.Body = nil
			//if err = tcpWriteProto(wr, hbProto); err != nil {
			//	log.Printf("key:%d tcpWriteProto() error(%v)", key, err)
			//	return
			//}
			//time.Sleep(heart)
		} else if proto.Operation == opHeartbeatReply {
			log.Printf("key:%d receive heartbeat", key)
			if err = conn.SetReadDeadline(time.Now().Add(heart + 60*time.Second)); err != nil {
				log.Printf("conn.SetReadDeadline() error(%v)", err)
				quit <- true
				return
			}
		} else {
			log.Printf("key:%d op:%d msg: %s", key, proto.Operation, string(proto.Body))
			atomic.AddInt64(&countDown, 1)
		}
	}
}

func tcpWriteProto(wr *bufio.Writer, proto *Proto) (err error) {
	// write
	if err = binary.Write(wr, binary.BigEndian, uint32(rawHeaderLen)+uint32(len(proto.Body))); err != nil {
		return
	}
	if err = binary.Write(wr, binary.BigEndian, rawHeaderLen); err != nil {
		return
	}
	if err = binary.Write(wr, binary.BigEndian, proto.Ver); err != nil {
		return
	}
	if err = binary.Write(wr, binary.BigEndian, proto.Operation); err != nil {
		return
	}
	if err = binary.Write(wr, binary.BigEndian, proto.Seq); err != nil {
		return
	}
	if proto.Body != nil {
		if err = binary.Write(wr, binary.BigEndian, proto.Body); err != nil {
			return
		}
	}
	err = wr.Flush()
	return
}

func tcpReadProto(rd *bufio.Reader, proto *Proto) (err error) {
	var (
		packLen   int32
		headerLen int16
	)
	// read
	if err = binary.Read(rd, binary.BigEndian, &packLen); err != nil {
		log.Fatal("心跳"+err.Error())
		return
	}
	if err = binary.Read(rd, binary.BigEndian, &headerLen); err != nil {
		log.Fatal("headerLen"+err.Error())
		return
	}
	if err = binary.Read(rd, binary.BigEndian, &proto.Ver); err != nil {
		log.Fatal("Ver"+err.Error())
		return
	}
	if err = binary.Read(rd, binary.BigEndian, &proto.Operation); err != nil {
		log.Fatal("Operation"+err.Error())
		return
	}
	if err = binary.Read(rd, binary.BigEndian, &proto.Seq); err != nil {
		log.Fatal("Seq"+err.Error())
		return
	}
	var (
		n, t    int
		bodyLen = int(packLen - int32(headerLen))
	)
	if bodyLen > 0 {
		proto.Body = make([]byte, bodyLen)
		for {
			if t, err = rd.Read(proto.Body[n:]); err != nil {
				return
			}
			if n += t; n == bodyLen {
				break
			}
		}
	} else {
		proto.Body = nil
	}
	return
}