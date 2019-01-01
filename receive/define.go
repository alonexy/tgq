package receive

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/google/uuid"
)

// Op Op
type Op [3]byte

func (p Op) String() string {
	switch p {
	case OpHeartbeat:
		return "心跳请求"
	case OpHeartbeatReply:
		return "心跳响应"
	case OpLogin:
		return "登陆请求"
	case OpLoginReply:
		return "登陆响应"
	case OpMessage:
		return "发送消息"
	case OpMessageReply:
		return "接收消息"
	default:
		return fmt.Sprintf("未知消息:%v", p)
	}
}

var (
	// OpHeartbeat  心跳请求
	OpHeartbeat Op = [3]byte{1, 1, 3}
	// OpHeartbeatReply 心跳响应
	OpHeartbeatReply Op = [3]byte{2, 1, 3}

	// OpLogin 登陆请求
	OpLogin Op = [3]byte{1, 2, 3}
	// OpLoginReply 登陆响应
	OpLoginReply Op = [3]byte{2, 2, 3}

	// OpMessage 发送消息
	OpMessage Op = [3]byte{1, 3, 3}
	// OpMessageReply 接收消息
	OpMessageReply Op = [3]byte{2, 3, 3}
)

// Header Header
type Header struct {
	Operate Op // 消息类型
	B256    byte
	C256    byte
	// DateSize int // 消息长度
}

func (p *Header) String() string {
	return fmt.Sprintf("header: %s, size:%d", p.Operate, p.DateSize())
}

// ReadFrom ReadFrom
func (p *Header) ReadFrom(r io.Reader) (n int64, err error) {
	buf := make([]byte, 5)

	var i int
	i, err = r.Read(buf)
	if err != nil {
		return
	}

	if i != 5 {
		return 0, errors.New("read header len not match")
	}

	p.Operate[0] = buf[0]
	p.Operate[1] = buf[1]
	p.Operate[2] = buf[2]
	p.B256 = buf[3]
	p.C256 = buf[4]

	return int64(i), nil
}

// WriteTo WriteTo
func (p *Header) WriteTo(w io.Writer) (n int64, err error) {

	var i int
	i, err = w.Write(p.Operate[:])
	if err != nil {
		return
	}

	_, err = w.Write([]byte{p.B256})
	if err != nil {
		return
	}

	i++

	_, err = w.Write([]byte{p.C256})
	if err != nil {
		return
	}
	i++

	return int64(i), nil

}

// DateSize DateSize
func (p *Header) DateSize() int {
	return int(p.B256)*256 + int(p.C256)
}

// Message Message
type Message struct {
	Header
	Data []byte
}

func (p *Message) String() string {
	return fmt.Sprintf("%s,data:%s", p.Header.String(), ToString(p.Data))
}

// WriteTo 发送消息
func (p *Message) WriteTo(w io.Writer) (n int64, err error) {
	n, err = p.Header.WriteTo(w)
	if err != nil {
		return
	}

	if len(p.Data) == 0 {
		return
	}

	var i int
	i, err = w.Write(p.Data)
	if err != nil {
		return
	}

	n += int64(i)

	return

}

// ReadFrom ReadFrom
func (p *Message) ReadFrom(r io.Reader) (n int64, err error) {
	n, err = p.Header.ReadFrom(r)
	if err != nil {
		return
	}

	var (
		i    int
		size = p.DateSize()
	)

	if size > 0 {
		p.Data = make([]byte, size)
		i, err = r.Read(p.Data)
		if err != nil {
			return
		}

		n += int64(i)
	}

	return
}

// NewMessage NewMessage
func NewMessage(op Op, data []byte) *Message {
	bodySize := len(data)

	msg := &Message{}
	msg.Header.Operate = op
	msg.Header.B256 = byte(bodySize / 256)
	msg.Header.C256 = byte(bodySize % 256)
	msg.Data = data

	return msg
}

// NewLoginMessage 登陆消息
func NewLoginMessage(user, password string) *Message {

	guid, _ := uuid.NewRandom()
	loginTime := time.Now().Format("2006-01-02 15:04:05")

	h := md5.New()
	h.Write([]byte(password + loginTime + user + guid.String()))
	token := hex.EncodeToString(h.Sum(nil))

	m := map[string]string{}
	m["UserName"] = user
	m["LoginTime"] = loginTime
	m["DevId"] = guid.String()
	m["token"] = token

	data, _ := json.Marshal(m)

	return NewMessage(OpLogin, data)
}

// NewQuoteMessage 请求行情消息
func NewQuoteMessage(types []string) *Message {

	m := map[string]interface{}{}
	m["Params"] = types
	m["RequestNo"] = "0"
	m["ServiceCode"] = "00001"

	data, _ := json.Marshal(m)

	return NewMessage(OpMessage, data)
}

// NewHeartbeatMessage 心跳请求包
func NewHeartbeatMessage() *Message {
	return NewMessage(OpMessage, nil)
}
