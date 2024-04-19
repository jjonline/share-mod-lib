package guzzle

import (
	"net/http"
	"time"
)

// Option 配置http.Client 属性 cookie jar/RoundTripper/Timeout
type Option func(*http.Client)

// CookieJar 设置cookie jar
func CookieJar(jar http.CookieJar) Option {
	return func(c *http.Client) {
		c.Jar = jar
	}
}

// Transport 设置RoundTripper
func Transport(transport http.RoundTripper) Option {
	return func(c *http.Client) {
		c.Transport = transport
	}
}

// Timeout 设置超时时间
func Timeout(timeout time.Duration) Option {
	return func(c *http.Client) {
		c.Timeout = timeout
	}
}

// WithOptions 重设http.Client相关配置
// 注意：此方法会新建guzzle与http.Client副本,不影响原来guzzle实例
//   - 调用方法，比如设置超时时间
//     result, err := client.Guzzle.WithOptions(guzzle.Timeout(12*time.Second)).Get(...)
//   - 可设置options如下:
//     guzzle.CookieJar(...)
//     guzzle.Transport(...)
//     guzzle.Timeout(...)
func (c *Client) WithOptions(options ...Option) *Client {
	//如果不传递配置,直接返回原副本
	if len(options) == 0 {
		return c
	}
	// 创建一个新的 guzzle 副本
	client := *c
	// 创建一个新的 httpClient 副本
	httpClient := *c.client
	for _, option := range options {
		option(&httpClient)
	}
	// 新的http client赋值给新的guzzle对像
	client.client = &httpClient
	return &client
}
