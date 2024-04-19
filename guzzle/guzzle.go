package guzzle

import (
	"context"
	"crypto/tls"
	"errors"
	"io"
	"net/http"
	"net/http/httptrace"
	"time"
)

// ErrResponseNotOK 当请求响应码非200时返回的错误
//   - 调用方只关注响应码为200的场景时，直接判断err是否为nil即可
//     result, err := client.JSON(xx,xx,xx)
//     if err != nil {
//     return
//     }
//     // your code // http响应码为200时的逻辑
//     ------------------------------------------------------------------------
//   - 调用方若需处理非200时返回值，如下处理：
//     if err != nil && errors.Is(err, guzzle.ErrResponseNotOK) {
//     // http响应码非200，此时result也是有值的
//     }
var ErrResponseNotOK = errors.New("failed response status code is not equal 200")

// defaultUserAgent 默认UA头，调用方法时可覆盖
var defaultUserAgent = "guzzle/go (module github.com/jjonline/share-mod-lib/guzzle)"

// Result 响应封装
type Result struct {
	StatusCode    int                   // 响应码
	ContentLength int64                 // 响应长度
	Header        http.Header           // 响应头
	Body          []byte                // 读取出来的响应body体字节内容
	TraceStack    func() []TraceItem    // trace调用栈
	TraceDuration func() *TraceDuration // trace调用时长统计
}

// Client http客户端相关方法封装
type Client struct {
	client      *http.Client
	hook        *RequestHookFunc
	enableTrace bool // 是否启用trace，默认禁用
}

type HookPayload struct {
	Request *http.Request `json:"request"`
	Result  Result        `json:"result"`
	Error   error         `json:"error"`
}

type RequestHookFunc func(*HookPayload)

// TraceGroup trace分组信息
type TraceGroup struct {
	GetConn              time.Time `json:"get_conn"`
	DNSStart             time.Time `json:"dns_start"`
	DNSDone              time.Time `json:"dns_done"`
	ConnectStart         time.Time `json:"connect_start"`
	ConnectDone          time.Time `json:"connect_done"`
	TLSHandshakeStart    time.Time `json:"tls_handshake_start"`
	TLSHandshakeDone     time.Time `json:"tls_handshake_done"`
	GotConn              time.Time `json:"got_conn"`
	WroteHeaders         time.Time `json:"wrote_headers"`
	WroteRequest         time.Time `json:"wrote_request"`
	GotFirstResponseByte time.Time `json:"got_first_response_byte"`
}

// TraceDuration trace时间
type TraceDuration struct {
	DNSLookup            time.Duration `json:"dns_lookup"`              // DNS查找 到 DNS查找结束
	Connect              time.Duration `json:"connect"`                 // 连接拨号时间：开始拨号 到 拨号成功
	TLSHandshake         time.Duration `json:"tls_handshake"`           // TLS握手时间：握手开始 到 握手结束
	GotConn              time.Duration `json:"got_conn"`                // 建立连接时间：开始连接 到 连接成功的时间
	GotFirstResponseByte time.Duration `json:"got_first_response_byte"` // 首包时间：开始连接 到 获取到响应首字节时间
	Total                time.Duration `json:"total"`                   // 总请求时间：开始连接 到 响应结束
}

type TraceItem struct {
	Key  string
	Time time.Time
}

// New 创建一个http客户端实例对象
//   - client *http.Client 可以自定义http请求的相关参数例如请求超时控制，使用默认则传 nil
func New(client *http.Client, hook *RequestHookFunc) *Client {
	if client == nil {
		client = http.DefaultClient
	}

	return &Client{
		client:      client,
		hook:        hook,
		enableTrace: false,
	}
}

// EnableTrace 启用
func (c *Client) EnableTrace() *Client {
	newC := *c
	httpClient := *c.client
	newC.client = &httpClient
	newC.enableTrace = true
	return &newC
}

// NewRequest 新建http请求，链式初始化请求，需链式 Do 方法才实际执行<可灵活自定义以实现诸如 http.MethodOptions 类型请求>
//   - method 请求方法：GET、POST等，使用 http.MethodGet http.MethodPost 等常量
//   - url    请求完整URL<可使用 guzzle.ToQueryURL 构造url里的query查询串>
//   - body   请求body体 io.Reader 类型
func (c *Client) NewRequest(ctx context.Context, method, url string, body io.Reader) (*http.Request, error) {
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}

	req.Close = true

	// 设置请求context
	req = req.WithContext(ctx)

	return req, nil
}

// Do 处理请求：用于链式调用
func (c *Client) Do(req *http.Request) (result Result, err error) {
	//触发自定义钩子
	defer func() {
		if c.hook != nil {
			hookData := &HookPayload{
				Request: req,
				Result:  result,
				Error:   err,
			}
			go (*c.hook)(hookData)
		}
	}()

	// set default user-agent if you do not set
	// header key is case-insensitive
	if req.Header.Get("User-Agent") == "" {
		req.Header.Set("User-Agent", defaultUserAgent)
	}

	res, err := c.client.Do(req)
	if err != nil {
		return result, err
	}

	defer func(Body io.ReadCloser) {
		_ = Body.Close()
	}(res.Body)

	row, err := io.ReadAll(res.Body)
	if err != nil {
		return result, err
	}

	// 非200时返回错误同时结果集仍然返回内容，以方便调用方需要处理状态码非200的场景
	if res.StatusCode != http.StatusOK {
		err = ErrResponseNotOK
	}

	// set result
	result.Body = row
	result.StatusCode = res.StatusCode
	result.Header = res.Header
	result.ContentLength = res.ContentLength

	return result, err
}

// Request 执行请求：实际执行请求<可灵活自定义以实现诸如 http.MethodOptions 类型请求>
//   - method 请求方法：GET、POST等，使用 http.MethodGet http.MethodPost 等常量
//   - url    请求完整URL<可使用 guzzle.ToQueryURL 构造url里的query查询串>
//   - body   请求body体 io.Reader 类型
//   - head   请求header部分
func (c *Client) Request(ctx context.Context, method, url string, body io.Reader, head map[string]string) (result Result, err error) {
	req, err := c.NewRequest(ctx, method, url, body)
	if err != nil {
		return
	}

	//获取trace信息
	req, completeFn := c.trace(ctx, req)
	defer func() {
		result.TraceStack, result.TraceDuration = completeFn(time.Now())
	}()

	for key, val := range head {
		req.Header.Add(key, val)
	}
	result, err = c.Do(req)
	return
}

// Get 执行 get 请求
//   - url    请求完整URL<可使用 guzzle.ToQueryURL 构造url里的query查询串>
//   - query  GET请求URl中的Query键值对，支持类型：map[string]string、map[string][]string<等价于 url.Values>，无则给 nil
//   - head   请求header部分键值对，无则给 nil
func (c *Client) Get(ctx context.Context, url string, query interface{}, head map[string]string) (result Result, err error) {
	req, err := c.NewRequest(ctx, http.MethodGet, ToQueryURL(url, query), nil)
	if err != nil {
		return Result{}, err
	}

	//获取trace信息
	req, completeFn := c.trace(ctx, req)
	defer func() {
		result.TraceStack, result.TraceDuration = completeFn(time.Now())
	}()

	for key, val := range head {
		req.Header.Add(key, val)
	}
	result, err = c.Do(req)
	return
}

// Delete 执行 delete 请求
//   - url    请求完整URL<可使用 guzzle.ToQueryURL 构造url里的query查询串>
//   - query  GET请求URl中的Query键值对，支持类型：map[string]string、map[string][]string<等价于 url.Values>，无则给 nil
//   - head   请求header部分键值对，无则给 nil
func (c *Client) Delete(ctx context.Context, url string, query interface{}, head map[string]string) (result Result, err error) {
	req, err := c.NewRequest(ctx, http.MethodDelete, ToQueryURL(url, query), nil)
	if err != nil {
		return Result{}, err
	}

	//获取trace信息
	req, completeFn := c.trace(ctx, req)
	defer func() {
		result.TraceStack, result.TraceDuration = completeFn(time.Now())
	}()

	for key, val := range head {
		req.Header.Add(key, val)
	}
	result, err = c.Do(req)
	return
}

// JSON 执行 post/put/patch/delete 请求，采用 json 格式<比较底层的方法>
//   - method 请求方法：GET、POST等，使用 http.MethodGet http.MethodPost 等常量
//   - url    请求完整URL<可使用 guzzle.ToQueryURL 构造url里的query查询串>
//   - body   请求body体 io.Reader 类型
//   - head   请求header部分键值对
func (c *Client) JSON(ctx context.Context, method, url string, body io.Reader, head map[string]string) (result Result, err error) {
	req, err := c.NewRequest(ctx, method, url, body)
	if err != nil {
		return
	}

	//获取trace信息
	req, completeFn := c.trace(ctx, req)
	defer func() {
		result.TraceStack, result.TraceDuration = completeFn(time.Now())
	}()

	for key, val := range head {
		req.Header.Add(key, val)
	}
	req.Header.Set("Content-Type", "application/json")

	result, err = c.Do(req)
	return
}

// Form 执行 post 请求，采用 form 表单格式<比较底层的方法>
//   - method 请求方法：GET、POST等，使用 http.MethodGet http.MethodPost 等常量
//   - url    请求完整URL<可使用 guzzle.ToQueryURL 构造url里的query查询串>
//   - body   请求body体 io.Reader 类型
//   - head   请求header部分键值对
func (c *Client) Form(ctx context.Context, method, url string, body io.Reader, head map[string]string) (result Result, err error) {
	req, err := c.NewRequest(ctx, method, url, body)
	if err != nil {
		return
	}

	//获取trace信息
	req, completeFn := c.trace(ctx, req)
	defer func() {
		result.TraceStack, result.TraceDuration = completeFn(time.Now())
	}()

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	for key, val := range head {
		req.Header.Add(key, val)
	}

	result, err = c.Do(req)
	return
}

// PostJSON 执行 post 请求，采用 json 格式
//   - url    请求完整URL<可使用 guzzle.ToQueryURL 构造url里的query查询串>
//   - body   请求body体，支持：字符串、字节数组、结构体等，最终会转换为 io.Reader 类型
//   - head   请求header部分键值对，无传nil
func (c *Client) PostJSON(ctx context.Context, url string, body interface{}, head map[string]string) (Result, error) {
	return c.JSON(ctx, http.MethodPost, url, ToJsonReader(body), head)
}

// PutJSON 执行 put 请求，采用 json 格式
//   - url    请求完整URL<可使用 guzzle.ToQueryURL 构造url里的query查询串>
//   - body   请求body体，支持：字符串、字节数组、结构体等；请传 guzzle.ToJsonReader 支持的参数类型
//   - head   请求header部分键值对，无传nil
func (c *Client) PutJSON(ctx context.Context, url string, body interface{}, head map[string]string) (Result, error) {
	return c.JSON(ctx, http.MethodPut, url, ToJsonReader(body), head)
}

// PatchJSON 执行 patch 请求，采用 json 格式
//   - url    请求完整URL<可使用 guzzle.ToQueryURL 构造url里的query查询串>
//   - body   请求body体，支持：字符串、字节数组、结构体等；请传 guzzle.ToJsonReader 支持的参数类型
//   - head   请求header部分键值对，无传nil
func (c *Client) PatchJSON(ctx context.Context, url string, body interface{}, head map[string]string) (Result, error) {
	return c.JSON(ctx, http.MethodPatch, url, ToJsonReader(body), head)
}

// DeleteJSON 执行 delete 请求，采用 json 格式
//   - url    请求完整URL<可使用 guzzle.ToQueryURL 构造url里的query查询串>
//   - body   请求body体，支持：字符串、字节数组、结构体等；请传 guzzle.ToJsonReader 支持的参数类型
//   - head   请求header部分键值对，无传nil
func (c *Client) DeleteJSON(ctx context.Context, url string, body interface{}, head map[string]string) (Result, error) {
	return c.JSON(ctx, http.MethodDelete, url, ToJsonReader(body), head)
}

// PostForm 执行 post 请求，采用 form 格式
//   - url    请求完整URL<可使用 guzzle.ToQueryURL 构造url里的query查询串>
//   - body   请求body体，支持：字符串、字节数组等；请传 guzzle.ToFormReader 支持的参数类型
//   - head   请求header部分键值对，无传nil
func (c *Client) PostForm(ctx context.Context, url string, body interface{}, head map[string]string) (Result, error) {
	return c.Form(ctx, http.MethodPost, url, ToFormReader(body), head)
}

// PutForm 执行 put 请求，采用 form 格式
//   - url    请求完整URL<可使用 guzzle.ToQueryURL 构造url里的query查询串>
//   - body   请求body体，支持：字符串、字节数组等；请传 guzzle.ToFormReader 支持的参数类型
//   - head   请求header部分键值对，无传nil
func (c *Client) PutForm(ctx context.Context, url string, body interface{}, head map[string]string) (Result, error) {
	return c.Form(ctx, http.MethodPut, url, ToFormReader(body), head)
}

// PatchForm 执行 patch 请求，采用 form 格式
//   - url    请求完整URL<可使用 guzzle.ToQueryURL 构造url里的query查询串>
//   - body   请求body体，支持：字符串、字节数组等；请传 guzzle.ToFormReader 支持的参数类型
//   - head   请求header部分键值对，无传nil
func (c *Client) PatchForm(ctx context.Context, url string, body interface{}, head map[string]string) (Result, error) {
	return c.Form(ctx, http.MethodPatch, url, ToFormReader(body), head)
}

// DeleteForm 执行 delete 请求，采用 form 格式
//   - url    请求完整URL<可使用 guzzle.ToQueryURL 构造url里的query查询串>
//   - body   请求body体，支持：字符串、字节数组等；请传 guzzle.ToFormReader 支持的参数类型
//   - head   请求header部分键值对，无传nil
func (c *Client) DeleteForm(ctx context.Context, url string, body interface{}, head map[string]string) (Result, error) {
	return c.Form(ctx, http.MethodDelete, url, ToFormReader(body), head)
}

// 调用顺序：可能有多次连接
//
//	-> GetConn -> DNSStart -> DNSDone -> ConnectStart -> ConnectDone
//	-> TLSHandshakeStart -> TLSHandshakeDone -> GotConn
//	-> WroteHeaders -> WroteRequest -> GotFirstResponseByte
func (c *Client) trace(ctx context.Context, req *http.Request) (
	newReq *http.Request,
	completeFn func(t time.Time) (func() []TraceItem, func() *TraceDuration)) {

	if !c.enableTrace {
		completeFn = func(t time.Time) (func() []TraceItem, func() *TraceDuration) {
			return func() []TraceItem {
					return nil
				}, func() *TraceDuration {
					return &TraceDuration{}
				}
		}
		return req, completeFn
	}

	//完成时间
	var completedAt time.Time

	//保存trace栈数据
	stackData := make([]TraceItem, 0)

	//完成后回调
	completeFn = func(t time.Time) (traceStackFn func() []TraceItem, traceDurationFn func() *TraceDuration) {
		completedAt = t

		//获取trace栈
		traceStackFn = func() []TraceItem {
			return stackData
		}

		//计算请求耗时
		traceDurationFn = func() *TraceDuration {
			return c.calcTraceDuration(completedAt, stackData)
		}
		return
	}

	//记录trace信息
	writeTrace := func(key string, t time.Time) {
		stackData = append(stackData, TraceItem{Key: key, Time: t})
	}

	trace := &httptrace.ClientTrace{
		GetConn: func(hostPort string) {
			writeTrace(GetConn, time.Now())
		},
		GotConn: func(info httptrace.GotConnInfo) {
			writeTrace(GotConn, time.Now())
		},
		GotFirstResponseByte: func() {
			writeTrace(GotFirstResponseByte, time.Now())
		},
		DNSStart: func(info httptrace.DNSStartInfo) {
			writeTrace(DNSStart, time.Now())
		},
		DNSDone: func(info httptrace.DNSDoneInfo) {
			writeTrace(DNSDone, time.Now())
		},
		ConnectStart: func(network, addr string) {
			writeTrace(ConnectStart, time.Now())
		},
		ConnectDone: func(network, addr string, err error) {
			writeTrace(ConnectDone, time.Now())
		},
		TLSHandshakeStart: func() {
			writeTrace(TLSHandshakeStart, time.Now())
		},
		TLSHandshakeDone: func(state tls.ConnectionState, err error) {
			writeTrace(TLSHandshakeDone, time.Now())
		},
		WroteHeaders: func() {
			writeTrace(WroteHeaders, time.Now())
		},
		WroteRequest: func(info httptrace.WroteRequestInfo) {
			writeTrace(WroteRequest, time.Now())
		},
	}

	// 设置请求context
	newReq = req.WithContext(httptrace.WithClientTrace(ctx, trace))

	return
}

func (c *Client) calcTraceDuration(completedAt time.Time, stackData []TraceItem) (duration *TraceDuration) {
	duration = &TraceDuration{}

	//数据分组：从GetConn到下一个GetConn为一组，可能有多组
	traceData := make([]*TraceGroup, 0, 3)
	var index int
	for _, msg := range stackData {
		if msg.Key == GetConn {
			traceData = append(traceData, &TraceGroup{})
		}

		index = len(traceData) - 1
		switch msg.Key {
		case GetConn:
			traceData[index].GetConn = msg.Time
		case DNSStart:
			traceData[index].DNSStart = msg.Time
		case DNSDone:
			traceData[index].DNSDone = msg.Time
		case ConnectStart:
			traceData[index].ConnectStart = msg.Time
		case ConnectDone:
			traceData[index].ConnectDone = msg.Time
		case TLSHandshakeStart:
			traceData[index].TLSHandshakeStart = msg.Time
		case TLSHandshakeDone:
			traceData[index].TLSHandshakeDone = msg.Time
		case GotConn:
			traceData[index].GotConn = msg.Time
		case WroteHeaders:
			traceData[index].WroteHeaders = msg.Time
		case WroteRequest:
			traceData[index].WroteRequest = msg.Time
		case GotFirstResponseByte:
			traceData[index].GotFirstResponseByte = msg.Time
		}
	}

	// 计算时间：
	// DNSLookup            time.Duration // DNS查找 到 DNS查找结束
	// Connect              time.Duration // 连接拨号时间：开始拨号 到 拨号成功
	// TLSHandshake         time.Duration // TLS握手时间：握手开始 到 握手结束
	// GotConn              time.Duration // 建立连接时间：开始连接 到 连接成功的时间
	// GotFirstResponseByte time.Duration // 首包时间：开始连接 到 获取到响应首字节时间
	// Total                time.Duration // 总请求时间：开始连接 到 响应结束
	for i := range traceData {
		if traceData[i].DNSDone.Unix() > 0 && traceData[i].DNSStart.Unix() > 0 {
			duration.DNSLookup += traceData[i].DNSDone.Sub(traceData[i].DNSStart)
		}
		if traceData[i].ConnectDone.Unix() > 0 && traceData[i].ConnectStart.Unix() > 0 {
			duration.Connect += traceData[i].ConnectDone.Sub(traceData[i].ConnectStart)
		}
		if traceData[i].TLSHandshakeDone.Unix() > 0 && traceData[i].TLSHandshakeStart.Unix() > 0 {
			duration.TLSHandshake += traceData[i].TLSHandshakeDone.Sub(traceData[i].TLSHandshakeStart)
		}
		if traceData[i].GotConn.Unix() > 0 && traceData[i].GetConn.Unix() > 0 {
			duration.GotConn += traceData[i].GotConn.Sub(traceData[i].GetConn)
		}
		if traceData[i].GotFirstResponseByte.Unix() > 0 && traceData[i].GetConn.Unix() > 0 {
			duration.GotFirstResponseByte += traceData[i].GotFirstResponseByte.Sub(traceData[i].GetConn)
		}
		if i == 0 {
			duration.Total += completedAt.Sub(traceData[i].GetConn)
		}
	}
	return
}
