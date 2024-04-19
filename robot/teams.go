package robot

import (
	"context"
	"fmt"
	"github.com/jjonline/share-mod-lib/guzzle"
	"net/http"
	"strings"
	"sync"
	"time"
)

// teams webhook文档地址：https://learn.microsoft.com/zh-cn/microsoftteams/platform/webhooks-and-connectors/how-to/add-incoming-webhook?tabs=dotnet

type teamsRobot struct {
	config      *config        // default config
	once        *config        // once config
	client      *guzzle.Client // guzzle客户端
	switchFunc  func() bool    // 开关函数，每次发送消息时触发：true-发送，false-不发送
	mutex       *sync.Mutex    // once mutex
	messageType string         // 消息类型：MessageCard/AdaptiveCard
}

// NewTeamsRobot
//   - webhook    Teams webhook
//   - secret     Teams webhook secret为空
//   - httpClient 自定义 *http.Client 可自主控制http请求客户端，给 nil 不则使用默认
//   - switchFunc 开关函数，返回true则真实发送，返回false则不真实发送/不用更改注释调用代码仅初始化时设置该值即可关闭真实发送逻辑
func NewTeamsRobot(webhook, secret string, httpClient *http.Client, switchFunc func() bool, mt string) Robot {
	return &teamsRobot{
		config: &config{
			webhook: webhook,
			secret:  secret,
		},
		once:        nil, // none once Config for init
		client:      guzzle.New(httpClient, nil),
		switchFunc:  switchFunc,
		mutex:       &sync.Mutex{},
		messageType: mt,
	}
}

// Once 配置单次<once>使用其他机器人
//   - webhook    Teams webhook
//   - secret     Teams webhook secret为空
func (s *teamsRobot) Once(webhook, secret string) Robot {
	s.once = &config{
		webhook: webhook,
		secret:  secret,
	}
	return s
}

// removeOnce 清理掉单次切换机器人配置
func (s *teamsRobot) removeOnce() {
	s.once = nil
}

// Info 提示（标题蓝色背景）
// title和content均支持emoji表情
func (s *teamsRobot) Info(ctx context.Context, title, markdownText string, t time.Time, links ...LinkItem) (err error) {
	params := s.buildParams(BgGreen, title, strings.TrimSuffix(markdownText, "\n")+
		"\n Time: "+t.In(UTCZone8Location).Format("2006-01-02 15:04:05"), links)
	return s.send(ctx, params)
}

// Warning 告警（标题黄色背景）
// title和content均支持emoji表情
func (s *teamsRobot) Warning(ctx context.Context, title, markdownText string, t time.Time, links ...LinkItem) (err error) {
	params := s.buildParams(BgYellow, title, strings.TrimSuffix(markdownText, "\n")+
		"\nTime: "+t.In(UTCZone8Location).Format("2006-01-02 15:04:05"), links)
	return s.send(ctx, params)
}

// Error 错误（标题红色背景）
// title和content均支持emoji表情
func (s *teamsRobot) Error(ctx context.Context, title, markdownText string, t time.Time, links ...LinkItem) (err error) {
	params := s.buildParams(BgRed, title, strings.TrimSuffix(markdownText, "\n")+
		"\nTime: "+t.In(UTCZone8Location).Format("2006-01-02 15:04:05"), links)
	return s.send(ctx, params)
}

// Message 提示（可指定主题颜色，使用已定义的常量）
// title和content均支持emoji表情
func (s *teamsRobot) Message(ctx context.Context, bg, title, markdownText string, t time.Time, links ...LinkItem) (err error) {
	params := s.buildParams(bg, title, strings.TrimSuffix(markdownText, "\n")+
		"\n Time: "+t.In(UTCZone8Location).Format("2006-01-02 15:04:05"), links)
	return s.send(ctx, params)
}

func (s *teamsRobot) buildParams(bg, title, markdownText string, links []LinkItem) interface{} {
	if s.messageType == MessageCard {
		linkNodes := make([]string, 0)
		for _, item := range links {
			linkNodes = append(linkNodes, fmt.Sprintf("[%s](%s)", item.Text, item.Url))
		}
		if len(linkNodes) > 0 {
			markdownText += "\n" + strings.Join(linkNodes, "     ")
		}

		return TeamsMessageCard{
			Type:       MessageCard,
			ThemeColor: TeamsThemeColorMapping[bg],
			Title:      title,
			Text:       strings.ReplaceAll(markdownText, "\n", "\n\n"), // \n需要两个才生效
		}
	} else {
		actions := make([]TeamsAdaptiveCardActionOpenUrl, 0)
		for _, item := range links {
			actions = append(actions, TeamsAdaptiveCardActionOpenUrl{
				Type:  "Action.OpenUrl",
				Title: item.Text,
				Url:   item.Url,
			})
		}

		body := []TeamsAdaptiveCardContainer{
			{
				Type: "Container",
				//Style: TeamsCardStyleMapping[bg],
				BackgroundImage: TeamsBgImageMapping[bg],
				Bleed:           true,
				Items: []TeamsAdaptiveCardTextBlock{
					{
						Type:    "TextBlock",
						Wrap:    true,
						Spacing: "none",
						Text:    title,
						Size:    "large",
						Weight:  "bolder",
						Color:   TeamsThemeTextColorMapping[bg],
					},
				},
			},
			{
				Type:  "Container",
				Bleed: true,
				Items: []TeamsAdaptiveCardTextBlock{
					{
						Type:    "TextBlock",
						Wrap:    true,
						Spacing: "none",
						Text:    strings.ReplaceAll(markdownText, "\n", "\n\n"),
						Size:    "default",
						Weight:  "default",
					},
				},
			},
		}

		return TeamsAdaptiveCard{
			Type: "message",
			Attachments: []TeamsAdaptiveCardAttachment{
				{
					ContentType: "application/vnd.microsoft.card.adaptive",
					Content: TeamsAdaptiveCardContent{
						Schema:  "http://adaptivecards.io/schemas/adaptive-card.json",
						Type:    AdaptiveCard,
						Version: "1.4",
						Body:    body,
						Actions: actions,
					},
				},
			},
		}
	}
}

// send 发送
func (s *teamsRobot) send(ctx context.Context, params interface{}) (err error) {
	if s.switchFunc != nil && !s.switchFunc() {
		return
	}

	// 获取webhook
	webhook, _ := s.getWebHookAndSecret()

	result, err := s.client.PostJSON(ctx, webhook, params, nil)
	if err != nil {
		return
	}

	if result.StatusCode == 200 && string(result.Body) == "1" {
		return nil
	}
	return fmt.Errorf("send msg err:(%d)%s", result.StatusCode, string(result.Body))
}

func (s *teamsRobot) getWebHookAndSecret() (webhook, secret string) {
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
