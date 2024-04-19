package robot

// 飞书消息类型
const (
	Text        = "text"        // 文本消息
	Interactive = "interactive" // 卡片消息
)

// teams消息类型
const (
	MessageCard  = "MessageCard"
	AdaptiveCard = "AdaptiveCard"
)

// 卡片标题颜色
const (
	BgGreen  = "green"  // 绿色
	BgYellow = "yellow" // 黄色
	BgRed    = "red"    // 红色
	BgBlue   = "blue"   // 蓝色
	BgPurple = "purple" // 紫色
	BgD9E36E = "D9E36E" // D9E36E
)

// TeamsThemeColorMapping Teams主题颜色
var TeamsThemeColorMapping = map[string]string{
	BgGreen:  "00FF00",
	BgYellow: "FFD700",
	BgRed:    "FF4040",
	BgBlue:   "0000FF",
	BgPurple: "9B30FF",
	BgD9E36E: "D9E36E",
}

// TeamsThemeTextColorMapping 标题字体颜色
var TeamsThemeTextColorMapping = map[string]string{
	BgGreen:  "",
	BgYellow: "",
	BgRed:    "light",
	BgBlue:   "light",
	BgPurple: "light",
	BgD9E36E: "",
}

//var TeamsCardStyleMapping = map[string]string{
//	BgGreen:  "good",
//	BgYellow: "warning",
//	BgRed:    "attention",
//}

var TeamsBgImageMapping = map[string]string{
	BgGreen:  "https://www.mytvsuper.com/tc/scoopplus/thumbor/VwCoh4ZdYcvJqEG-PmMpfMsI24I=/0x0/public/images/202303/fa9c61f0-6de4-4828-a9fe-2ea3c97358fa.png",
	BgYellow: "https://www.mytvsuper.com/tc/scoopplus/thumbor/jraG-cDg-9xUTnszo-TaAUxD_Do=/0x0/public/images/202303/598ae085-ef3c-4b54-bb1c-6e473fb58533.png",
	BgRed:    "https://www.mytvsuper.com/tc/scoopplus/thumbor/JcQmDUR1At4F77-5QVVW4jFVPLo=/0x0/public/images/202303/76458072-b896-4bb1-8403-c60bb2a01b5b.png",
	BgBlue:   "https://www.mytvsuper.com/tc/scoopplus/thumbor/hWkvVFKo9S5oEv52drwSKajj7yQ=/0x0/public/images/202303/27a9a836-83e9-47c7-bc22-506a0e95b677.png",
	BgPurple: "https://www.mytvsuper.com/tc/scoopplus/thumbor/S_WTr14jbDIVgJmSuE3eg5Lj8Fs=/0x0/public/images/202303/f95b934b-ff62-401c-8fcd-7dd21a54085a.png",
	BgD9E36E: "https://www.mytvsuper.com/tc/scoopplus/thumbor/_B7qqy_IC4bzuEsJVCZ7d_aufeo=/0x0/public/images/202303/b10014f1-b096-41d6-a1ce-a0e45fd63f07.png",
}

type config struct {
	webhook string // webhook
	secret  string // 秘钥
}

type CardMsgParams struct {
	Timestamp int64    `json:"timestamp"`
	Sign      string   `json:"sign"`
	MsgType   string   `json:"msg_type"`
	Card      CardItem `json:"card"`
}

type CardItem struct {
	Config   CardConfigItem    `json:"config"`
	Header   CardHeaderItem    `json:"header"`
	Elements []CardElementItem `json:"elements"`
}

type CardConfigItem struct {
	WideScreenMode bool `json:"wide_screen_mode"` // true
	EnableForward  bool `json:"enable_forward"`   // true
}

type CardHeaderItem struct {
	Title    CardHeaderTitleItem `json:"title"`
	Template string              `json:"template"` // 卡片标题颜色：blue red
}

type CardHeaderTitleItem struct {
	Content string `json:"content"` // 卡片标题
	Tag     string `json:"tag"`     // plain_text
}

type CardElementItem struct {
	Tag     string `json:"tag"`     // markdown
	Content string `json:"content"` // markdown内容
}

type SendResponse struct {
	StatusCode    int    `json:"StatusCode"`    // 成功：StatusCode=0
	StatusMessage string `json:"StatusMessage"` // 成功：StatusMessage=success
	Code          int    `json:"code"`          // 失败错误码
	Msg           string `json:"msg"`           // 失败错误信息
}

type TeamsMessageCard struct {
	Type       string `json:"@type"`      // 类型：MessageCard
	ThemeColor string `json:"themeColor"` // 颜色，可自定义，例子：FF4040
	Title      string `json:"title"`      // 标题，支持markdown和emoji表情
	Text       string `json:"text"`       // 内容，支持markdown和emoji表情
}

// TeamsAdaptiveCard 文档：https://adaptivecards.io/explorer/AdaptiveCard.html
type TeamsAdaptiveCard struct {
	Type        string                        `json:"type"`
	Attachments []TeamsAdaptiveCardAttachment `json:"attachments"`
}

type TeamsAdaptiveCardAttachment struct {
	ContentType string                   `json:"contentType"` //传：application/vnd.microsoft.card.adaptive
	Content     TeamsAdaptiveCardContent `json:"content"`
}

type TeamsAdaptiveCardContent struct {
	Schema  string                           `json:"$schema"` //传：http://adaptivecards.io/schemas/adaptive-card.json
	Type    string                           `json:"type"`    //传：AdaptiveCard
	Version string                           `json:"version"` //传：1.4
	Body    []TeamsAdaptiveCardContainer     `json:"body"`
	Actions []TeamsAdaptiveCardActionOpenUrl `json:"actions"` //暂时仅支持跳转链接
}

// TeamsAdaptiveCardContainer 文档：https://adaptivecards.io/explorer/Container.html
type TeamsAdaptiveCardContainer struct {
	Type            string                       `json:"type"`                      //类型：Container
	Style           string                       `json:"style,omitempty"`           //样式：default/good/warning/attention
	BackgroundImage string                       `json:"backgroundImage,omitempty"` //背景图片url
	Bleed           bool                         `json:"bleed"`                     //填充：是否跟随父元素的填充
	Items           []TeamsAdaptiveCardTextBlock `json:"items"`
}

// TeamsAdaptiveCardTextBlock 文档：https://adaptivecards.io/explorer/TextBlock.html
type TeamsAdaptiveCardTextBlock struct {
	Type    string `json:"type"`             //类型：TextBlock
	Wrap    bool   `json:"wrap"`             //是否完整显示文本
	Spacing string `json:"spacing"`          //与上个容器间隔大小
	Text    string `json:"text"`             //文本内容，支持markdown
	Size    string `json:"size,omitempty"`   //文本大小：default/small/medium/large/extraLarge
	Weight  string `json:"weight,omitempty"` //文本样式：default/lighter/bolder
	Color   string `json:"color,omitempty"`  //文本颜色：good/warning/attention
}

// TeamsAdaptiveCardActionOpenUrl 文档：https://adaptivecards.io/explorer/Action.OpenUrl.html
type TeamsAdaptiveCardActionOpenUrl struct {
	Type  string `json:"type"`  //Action.OpenUrl
	Title string `json:"title"` //按钮文本
	Url   string `json:"url"`   //链接
}

type LinkItem struct {
	Text string `json:"text"`
	Url  string `json:"url"`
}
