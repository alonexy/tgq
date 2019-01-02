package receive

import (
	"bufio"
	"context"
	"io"

	"net"
	"time"

	log "github.com/golang/glog"
)

// Client Client
type Client struct {
	opts        Options
	ctx         context.Context
	conn        net.Conn
	retryN      int
	requestChan chan io.WriterTo
	resultChan  chan *Message
}

// New New
func New(opts ...Option) *Client {
	p := &Client{}

	for _, o := range opts {
		o(&p.opts)
	}

	p.init()

	p.requestChan = make(chan io.WriterTo, 10)
	p.resultChan = make(chan *Message, 10)

	return p
}

func (p *Client) init() {

	if p.opts.Ctx == nil {
		p.opts.Ctx = context.Background()
	}

	if p.opts.Timeout == 0 {
		p.opts.Timeout = time.Second * 10
	}

	if p.opts.Receiver == nil {
		p.opts.Receiver = func(msg *Message) {
			p.opts.Logger.Println("receive message: ", msg)
		}
	}

	log.Infoln("init opts: ", p.opts)

}

func (p *Client) connect(addr string) error {
	conn, err := net.DialTimeout("tcp", addr, p.opts.Timeout)
	if err != nil {
		return err
	}

	p.conn = conn
	return nil
}

// 重试等待时间  大于N时间一重新开始
func (p *Client) backof() time.Duration {

	if p.retryN > 10 {
		p.retryN = 10
	}

	return time.Second * 2 * time.Duration(p.retryN)
}

func (p *Client) cleanChan() {
	for {
		select {
		case <-p.requestChan:
		case <-p.resultChan:
		default:
			return
		}
	}
}

func (p *Client) closeConn() {
	if p.conn != nil {
		p.conn.Close()
	}
}

// Start Start
func (p *Client) Start() {
	log.Infoln("client start")
	for {

		select {
		case <-p.opts.Ctx.Done():
			log.Warningln("client ctx.Done exit")
			return
		default:
		}

		p.cleanChan()
		// 建立连接
		log.Infoln("start to connect", p.opts.Addrs)

		err := p.connect(p.opts.Addrs)
		if err != nil {
			p.retryN++
			// 失败，等待重连
			log.Warningf("connect faild err:%v sleep %d to retry\n", err, p.backof())
			time.Sleep(p.backof())
			continue
		}

		log.Infoln("connect ok")

		ctx, cancel := context.WithCancel(p.opts.Ctx)

		go p.heartBeat(ctx, cancel)

		go p.handleResult(ctx, cancel)

		go p.handleReqeust(ctx, cancel)

		go p.ReceiveResult(ctx, cancel)

		if err := p.login(); err != nil {
			log.Errorf("login faild err:%v sleep %d to reconnect \n", err, p.backof())
			time.Sleep(p.backof())
			cancel()
			continue
		}

		select {
		case <-ctx.Done():
			cancel()
		}

		p.closeConn()

	}

}

func (p *Client) login() error {
	_, err := NewLoginMessage(p.opts.User, p.opts.Password).WriteTo(p.conn)
	if err != nil {
		return err
	}

	return nil
}

func (p *Client) heartBeat(ctx context.Context, cancel context.CancelFunc) {
	t := time.NewTicker(p.opts.Timeout)
	for {
		select {
		case <-ctx.Done():
			log.Warningln("heartBeat ctx.Done exit")
			return
		case <-t.C:
			log.Infoln("heartBeat  tick send heartbeat")
			p.sendRequest(NewMessage(OpHeartbeat, nil))
		}
	}
}

func (p *Client) handleReqeust(ctx context.Context, cancel context.CancelFunc) {
	for {
		select {
		case <-ctx.Done():
			log.Warningln("handleReqeust ctx.Done exit")
			return
		case req, ok := <-p.requestChan:
			if !ok {
				log.Warningln("handleReqeust requestChan close exit")
				return
			}

			log.Infoln("handleReqeust rev request: ", req)

			// 发送请求失败，断开连接
			_, err := req.WriteTo(p.conn)
			if err != nil {
				log.Errorln("handleReqeust request WriteTo err: ", err)
				cancel()
				return
			}
		}
	}
}

func (p *Client) handleResult(ctx context.Context, cancel context.CancelFunc) {
	rd := bufio.NewReader(p.conn)
	for {

		select {
		case <-ctx.Done():
			log.Warningln("handleResult ctx.Done exit")
			return
		default:
		}

		msg := &Message{}

		_, err := msg.ReadFrom(rd)
		if err != nil {
			log.Errorln("handleResult readFrom err: ", err)
			cancel()
			return
		}

		switch msg.Operate {
		case OpLoginReply: // 登陆响应
			log.Infoln("handleResult login ok")
			go p.requestQuote()
			continue
		case OpHeartbeatReply:
			log.Infoln("handleResult rev heatbeat reply ok")
			continue
		case OpHeartbeat: // 心跳请求
			log.Infoln("handleResult rev heartbeat from server, send reply")
			p.sendRequest(NewMessage(OpHeartbeatReply, nil))
			continue
		default:
			log.Infoln("handleResult rev message", ToString(msg.Data))
			p.resultChan <- msg
		}

	}
}

// sendRequest sendRequest
func (p *Client) sendRequest(req io.WriterTo) {
	p.requestChan <- req
}

// ReceiveResult ReceiveResult
func (p *Client) ReceiveResult(ctx context.Context, cancel context.CancelFunc) {
	for {
		select {
		case <-ctx.Done():
			log.Warningln("ReceiveResult ctx.Done exit")
			return
		case msg := <-p.resultChan:
			p.opts.Receiver(msg)
		}
	}
}

func (p *Client) requestQuote() {
	if len(p.opts.Quotes) > 0 {
		p.requestChan <- NewQuoteMessage(p.opts.Quotes)
	}
}
