package email

const (
	ContentTypePlainText = "text/plain" // 纯文本
	ContentTypeHtml      = "text/html"  // html
)

type sendGridConfig struct {
	host     string // sendgrid host：https://api.sendgrid.com
	endpoint string // sendgrid api：/v3/mail/send
	key      string // sendgrid key
}

type smtpConfig struct {
	host     string
	port     int
	userName string
	password string
}

type SendParams struct {
	From        string       `json:"from"`         // 发件人
	To          []string     `json:"to"`           // 收件人，支持多个
	Subject     string       `json:"subject"`      // 主题
	ContentType string       `json:"content_type"` // 邮件内容类型
	Content     string       `json:"content"`      // 邮件内容
	Attachments []Attachment `json:"attachments"`  // 附件，支持多个
}

type Attachment struct {
	Filename string `json:"filename"` // 文件名
	Type     string `json:"type"`     // 文件类型，默认：application/json
	Content  []byte `json:"content"`  // 文件内容
}
