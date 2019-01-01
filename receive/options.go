package receive

import (
	"context"
	"log"
	"time"
)

// Option 修改参数项
type Option func(*Options)

// Options 参数
type Options struct {
	Addrs    string // 请求地址
	User     string
	Password string
	Quotes   []string
	Ctx      context.Context
	Timeout  time.Duration
	Logger   *log.Logger
	Receiver func(*Message)
}

// WithCtx WithCtx
func WithCtx(ctx context.Context) Option {
	return func(o *Options) {
		o.Ctx = ctx
	}
}

// WithAddrs 请求地址
func WithAddrs(addr string) Option {
	return func(o *Options) {
		o.Addrs = addr
	}
}

// WithAuth 登陆账号密码参数
func WithAuth(user, password string) Option {
	return func(o *Options) {
		o.User = user
		o.Password = password
	}
}

// WithQuoteTypes 请求行情代码列表
func WithQuoteTypes(types []string) Option {
	return func(o *Options) {
		o.Quotes = types
	}
}

// WithTimeout 连接超时
func WithTimeout(d time.Duration) Option {
	return func(o *Options) {
		o.Timeout = d
	}
}

// WithReceiver WithReceiver
func WithReceiver(f func(*Message)) Option {
	return func(o *Options) {
		o.Receiver = f
	}
}
