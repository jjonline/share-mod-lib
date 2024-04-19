package email

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"github.com/sendgrid/rest"
	"github.com/sendgrid/sendgrid-go"
	"github.com/sendgrid/sendgrid-go/helpers/mail"
	"net/http"
)

type Email struct {
	config *sendGridConfig
	once   *sendGridConfig
}

// New 初始化一个基于sendGrid的邮件发送实例
//   - host     SendGrid的host
//   - endpoint SendGrid的endpoint即API
//   - key	    SendGrid的秘钥
func New(host, endpoint, key string) *Email {
	return &Email{
		config: &sendGridConfig{
			host: host, endpoint: endpoint, key: key,
		},
		once: nil,
	}
}

// Once 配置单次使用的配置
func (s *Email) Once(host, endpoint, key string) *Email {
	s.once = &sendGridConfig{
		host: host, endpoint: endpoint, key: key,
	}
	return s
}

// removeOnce 清理掉单次切换
func (s *Email) removeOnce() {
	s.once = nil
}

// Send 发送使用sendgrid接口的邮件
//   - ctx context
//   - params 拟发送邮件的结构体参数
func (s *Email) Send(ctx context.Context, params SendParams) (resp *rest.Response, err error) {
	defer s.removeOnce() // remove once config anyway

	if params.From == "" || len(params.To) == 0 || params.Content == "" ||
		(params.ContentType != ContentTypePlainText && params.ContentType != ContentTypeHtml) {
		err = errors.New("param error")
		return
	}

	// 发件人
	m := mail.NewV3Mail()
	m.Subject = params.Subject
	m.SetFrom(mail.NewEmail(params.From, params.From))

	// 收件人
	to := make([]*mail.Email, 0)
	for _, addr := range params.To {
		to = append(to, mail.NewEmail(addr, addr))
	}
	p := mail.NewPersonalization()
	p.AddTos(to...)
	m.AddPersonalizations(p)

	// 添加内容
	m.AddContent(mail.NewContent(params.ContentType, params.Content))

	// 添加附件
	attachments := make([]*mail.Attachment, 0)
	for _, attach := range params.Attachments {
		attachments = append(attachments, mail.NewAttachment().
			SetFilename(BEncoding(attach.Filename)).
			SetType("application/json").
			SetContent(base64.StdEncoding.EncodeToString(attach.Content)))
	}
	m.AddAttachment(attachments...)

	// check use once config
	cnf := s.config
	if s.once != nil {
		cnf = s.once
	}

	req := sendgrid.GetRequest(cnf.key, cnf.endpoint, cnf.host)
	req.Method = http.MethodPost
	req.Body = mail.GetRequestBody(m)
	return sendgrid.API(req)
}

// BEncoding 附件名转base64
func BEncoding(s string) string {
	return fmt.Sprintf("%s%s%s", "=?UTF-8?b?", base64.StdEncoding.EncodeToString([]byte(s)), "?=")
}
