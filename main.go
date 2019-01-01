package main

import (
	"context"
	"flag"

	"os"
	"os/signal"
	"runtime"
	"sync"
	"syscall"

	api "github.com/cenwj/quote/api/comet/grpc"
	"github.com/cenwj/quote/comet"
	"github.com/cenwj/quote/conf"
	"github.com/cenwj/quote/pkg/bytes"
	"github.com/cenwj/quote/receive"
	log "github.com/golang/glog"
)

var (
	srv *comet.Server
)

func main() {

	flag.Parse()
	if err := conf.Init(); err != nil {
		panic(err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	wg := &WaitGroupWrapper{}

	signChan := make(chan os.Signal)

	signal.Notify(signChan, syscall.SIGHUP, syscall.SIGQUIT, syscall.SIGTERM, syscall.SIGINT)

	wg.Wrap(func() {
		for {
			select {
			case <-ctx.Done():
				log.Warningln("s.exit")
				return
			case s := <-signChan:

				switch s {
				case syscall.SIGQUIT, syscall.SIGTERM, syscall.SIGINT:
					// 收到退出signal是地关闭 exitChan
					// fmt.Println("s.signal", s)
					cancel()
					return
				case syscall.SIGHUP:
				default:
					// fmt.Println("s.signal default", s)
					return
				}
			}

		}
	})

	wg.Wrap(func() {
		log.Infoln("goim-comet start")
		srv = comet.NewServer(conf.Conf)
		if err := comet.InitWhitelist(conf.Conf.Whitelist); err != nil {
			log.Errorln("comet InitWhitelist err", err)
			cancel()
			return
		}
		if err := comet.InitTCP(srv, conf.Conf.TCP.Bind, conf.Conf.MaxProc); err != nil {
			log.Errorln("comet InitTCP err", err)
			cancel()
			return
		}
		if err := comet.InitWebsocket(srv, conf.Conf.WebSocket.Bind, conf.Conf.MaxProc); err != nil {
			log.Errorln("comet InitWebsocket err", err)
			cancel()
			return
		}
		if conf.Conf.WebSocket.TLSOpen {
			if err := comet.InitWebsocketWithTLS(srv, conf.Conf.WebSocket.TLSBind, conf.Conf.WebSocket.CertFile, conf.Conf.WebSocket.PrivateFile, runtime.NumCPU()); err != nil {
				log.Errorln("comet InitWebsocketWithTLS err", err)
				cancel()
				return
			}
		}

		// rpcSrv := grpc.New(conf.Conf.RPCServer, srv)

		// log.Infoln("comet rpc start listen:", conf.Conf.RPCServer.Addr)

		select {
		case <-ctx.Done():
			// rpcSrv.GracefulStop()
			srv.Close()
		}
	})

	wg.Wrap(func() {

		c := receive.New(
			receive.WithAuth(conf.Conf.Quote.User, conf.Conf.Quote.Password),
			receive.WithAddrs(conf.Conf.Quote.Addr),
			receive.WithQuoteTypes(conf.Conf.Quote.Quotes),
			receive.WithCtx(ctx),
			receive.WithReceiver(func(msg *receive.Message) {

				if srv == nil {
					return
				}

				body := msg.Data

				buf := bytes.NewWriterSize(len(body) + 64)
				p := &api.Proto{
					Ver:  1,
					Op:   api.OpSendMsg,
					Body: body,
				}
				p.WriteTo(buf)
				p.Body = buf.Buffer()
				p.Op = api.OpRaw
				go func() {
					for _, bucket := range srv.Buckets() {
						bucket.Broadcast(p, 1000) // 1000 与客户端连接时accept值相同
					}
				}()

			}),
		)

		c.Start()
	})

	wg.Wait()
	log.Infof("goim-comet  exit")
	log.Flush()
}

// WaitGroupWrapper WaitGroupWrapper
type WaitGroupWrapper struct {
	sync.WaitGroup
}

// Wrap Wrap
func (w *WaitGroupWrapper) Wrap(fn func()) {
	w.Add(1)
	go func() {
		defer w.Done()
		fn()
	}()
}
