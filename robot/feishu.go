package robot

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/jjonline/share-mod-lib/guzzle"
	"net/http"
	"strings"
	"sync"
	"time"
)

// 飞书机器人文档地址：https://www.feishu.cn/hc/zh-CN/articles/360024984973
// 飞书机器人文档地址：https://open.feishu.cn/document/ukTMukTMukTM/ucTM5YjL3ETO24yNxkjN
// 飞书机器人markdown说明：https://open.feishu.cn/document/ukTMukTMukTM/uADOwUjLwgDM14CM4ATN

type feishuRobot struct {
	config     *config        // default config
	once       *config        // once config
	client     *guzzle.Client // guzzle客户端
	switchFunc func() bool    // 开关函数，每次发送消息时触发：true-发送，false-不发送
	mutex      *sync.Mutex    // once mutex
}

// NewFeishuRobot
//   - webhook    飞书webhook
//   - secret     飞书webhook对应的秘钥
//   - httpClient 自定义 *http.Client 可自主控制http请求客户端，给 nil 不则使用默认
//   - switchFunc 开关函数，返回true则真实发送，返回false则不真实发送/不用更改注释调用代码仅初始化时设置该值即可关闭真实发送逻辑
func NewFeishuRobot(webhook, secret string, httpClient *http.Client, switchFunc func() bool) Robot {
	return &feishuRobot{
		config: &config{
			webhook: webhook,
			secret:  secret,
		},
		once:       nil, // none once Config for init
		client:     guzzle.New(httpClient, nil),
		switchFunc: switchFunc,
		mutex:      &sync.Mutex{},
	}
}

// Once 配置单次<once>使用其他飞书机器人
//   - webhook    飞书webhook
//   - secret     飞书webhook对应的秘钥
func (s *feishuRobot) Once(webhook, secret string) Robot {
	s.once = &config{
		webhook: webhook,
		secret:  secret,
	}
	return s
}

// removeOnce 清理掉单次切换机器人配置
func (s *feishuRobot) removeOnce() {
	s.once = nil
}

// Info 提示（标题蓝色背景）
// title和content均支持emoji表情
// markdown写法说明：https://open.feishu.cn/document/ukTMukTMukTM/uADOwUjLwgDM14CM4ATN
func (s *feishuRobot) Info(ctx context.Context, title, markdownText string, t time.Time, links ...LinkItem) (err error) {
	params := s.buildParams(BgGreen, title, strings.TrimSuffix(markdownText, "\n")+
		"\nTime: "+t.In(UTCZone8Location).Format("2006-01-02 15:04:05"), links)
	return s.send(ctx, params)
}

// Warning 告警（标题黄色背景）
// title和content均支持emoji表情
// markdown写法说明：https://open.feishu.cn/document/ukTMukTMukTM/uADOwUjLwgDM14CM4ATN
func (s *feishuRobot) Warning(ctx context.Context, title, markdownText string, t time.Time, links ...LinkItem) (err error) {
	params := s.buildParams(BgYellow, title, strings.TrimSuffix(markdownText, "\n")+
		"\nTime: "+t.In(UTCZone8Location).Format("2006-01-02 15:04:05"), links)
	return s.send(ctx, params)
}

// Error 错误（标题红色背景）
// title和content均支持emoji表情
// markdown写法说明：https://open.feishu.cn/document/ukTMukTMukTM/uADOwUjLwgDM14CM4ATN
func (s *feishuRobot) Error(ctx context.Context, title, markdownText string, t time.Time, links ...LinkItem) (err error) {
	params := s.buildParams(BgRed, title, strings.TrimSuffix(markdownText, "\n")+
		"\nTime: "+t.In(UTCZone8Location).Format("2006-01-02 15:04:05"), links)
	return s.send(ctx, params)
}

// Message 提示（可指定主题颜色，使用已定义的常量）
// title和content均支持emoji表情
func (s *feishuRobot) Message(ctx context.Context, bg, title, markdownText string, t time.Time, links ...LinkItem) (err error) {
	params := s.buildParams(bg, title, strings.TrimSuffix(markdownText, "\n")+
		"\nTime: "+t.In(UTCZone8Location).Format("2006-01-02 15:04:05"), links)
	return s.send(ctx, params)
}

func (s *feishuRobot) buildParams(bg, title, markdownText string, links []LinkItem) CardMsgParams {
	linkNodes := make([]string, 0)
	for _, item := range links {
		linkNodes = append(linkNodes, fmt.Sprintf("[%s](%s)", item.Text, item.Url))
	}
	if len(linkNodes) > 0 {
		markdownText += "\n" + strings.Join(linkNodes, "    ")
	}

	return CardMsgParams{
		MsgType: Interactive,
		Card: CardItem{
			Config: CardConfigItem{
				WideScreenMode: true,
				EnableForward:  true,
			},
			Header: CardHeaderItem{
				Title: CardHeaderTitleItem{
					Content: title,
					Tag:     "plain_text",
				},
				Template: bg,
			},
			Elements: []CardElementItem{
				{
					Tag:     "markdown",
					Content: markdownText,
				},
			},
		},
	}
}

// send 发送
func (s *feishuRobot) send(ctx context.Context, params CardMsgParams) (err error) {
	if s.switchFunc != nil && !s.switchFunc() {
		return
	}

	// 获取webhook、secret
	webhook, secret := s.getWebHookAndSecret()

	now := time.Now().Unix()
	sign, err := s.sign(now, secret)
	if err != nil {
		return fmt.Errorf("sign err:%s", err.Error())
	}

	params.Sign = sign
	params.Timestamp = now
	result, err := s.client.PostJSON(ctx, webhook, params, nil)
	if err != nil {
		return
	}

	var resp SendResponse
	if err = json.Unmarshal(result.Body, &resp); err != nil {
		return
	}

	if resp.StatusCode == 0 && resp.Code == 0 {
		return nil
	}
	return fmt.Errorf("send msg err:(%d)%s", resp.Code, resp.Msg)
}

func (s *feishuRobot) getWebHookAndSecret() (webhook, secret string) {
	webhook, secret = s.config.webhook, s.config.secret
	if s.once != nil && s.mutex.TryLock() {
		defer s.mutex.Unlock()
		//防止并发
		if s.once != nil {
			webhook, secret = s.once.webhook, s.once.secret
		}
		s.removeOnce()
	}
	return
}

// sign 签名：timestamp + key 做sha256, 再进行base64 encode
func (s *feishuRobot) sign(timestamp int64, secret string) (string, error) {
	stringToSign := fmt.Sprintf("%v", timestamp) + "\n" + secret
	var data []byte
	h := hmac.New(sha256.New, []byte(stringToSign))
	_, err := h.Write(data)
	if err != nil {
		return "", err
	}
	signature := base64.StdEncoding.EncodeToString(h.Sum(nil))
	return signature, nil
}
