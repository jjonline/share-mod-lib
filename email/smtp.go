package email

import (
	"context"
	"errors"
	mail "github.com/xhit/go-simple-mail/v2"
)

type Smtp struct {
	server     *mail.SMTPServer
	onceServer *mail.SMTPServer
	config     *smtpConfig
	once       *smtpConfig
}

// NewSmtp 实例化一个smtp发送邮件实例
//   - host  	smtp-server地址，例如 smtp.qq.com
//   - port	    smtp-server端口，例如 587
//   - userName smtp-server登录用户名
//   - password smtp-server登录用户密码
func NewSmtp(host string, port int, userName, password string) *Smtp {
	// init server
	server := mail.NewSMTPClient()
	server.Host = host
	server.Port = port
	server.Username = userName
	server.Password = password
	server.Authentication = mail.AuthPlain // 默认认证方式-绝大多数认证都是这个方式

	return &Smtp{
		server:     server,
		onceServer: nil,
		config: &smtpConfig{
			host:     host,
			port:     port,
			userName: userName,
			password: password,
		},
		once: nil,
	}
}

// UseServer 使用底层smtp对象进行设置操作
func (s *Smtp) UseServer() *mail.SMTPServer {
	if s.onceServer != nil {
		return s.onceServer
	}
	return s.server
}

// Once 配置单次使用的配置
func (s *Smtp) Once(host string, port int, userName, password string) *Smtp {
	s.once = &smtpConfig{
		host:     host,
		port:     port,
		userName: userName,
		password: password,
	}

	// init server
	server := mail.NewSMTPClient()
	server.Host = host
	server.Port = port
	server.Username = userName
	server.Password = password
	server.Authentication = mail.AuthPlain // 默认认证方式-绝大多数认证都是这个方式

	s.onceServer = server

	return s
}

// removeOnce 清理掉单次切换
func (s *Smtp) removeOnce() {
	s.once = nil
	s.onceServer = nil
}

// Send 执行smtp邮件发送
//   - ctx context
//   - params 拟发送邮件的结构体参数
func (s *Smtp) Send(ctx context.Context, params SendParams) (err error) {
	// support once server
	server := s.server
	if s.once != nil {
		server = s.onceServer
	}

	client, err := server.Connect()
	if err != nil {
		return err
	}

	// contentType
	if params.ContentType != ContentTypePlainText && params.ContentType != ContentTypeHtml {
		return errors.New("param error")
	}

	// detect mime type
	cType := mail.TextCalendar
	if ContentTypeHtml == params.ContentType {
		cType = mail.TextHTML
	}

	msg := mail.NewMSG()
	msg.SetFrom(params.From).AddTo(params.To...).SetSubject(params.Subject).SetBody(cType, params.Content)

	// attach file
	if len(params.Attachments) > 0 {
		for _, att := range params.Attachments {
			msg.Attach(&mail.File{
				FilePath: "",
				Name:     att.Filename,
				MimeType: att.Type,
				B64Data:  "",
				Data:     att.Content,
				Inline:   false,
			})
		}
	}

	return msg.Send(client)
}
