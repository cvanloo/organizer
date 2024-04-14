package organizer

import (
	"bytes"
	"gopkg.in/gomail.v2"
	"text/template"
)

var tmplLoginLink = template.Must(template.New("LoginLink").Parse(loginLinkBody))

type (
	Mailer struct{
		Dialer *gomail.Dialer
		ThisSender string
	}
	MailConfig struct{
		Host string
		Port int
		Username, Password string
		ThisSender string
	}
)

func NewMailer(cfg MailConfig) *Mailer {
	d := gomail.NewDialer(cfg.Host, cfg.Port, cfg.Username, cfg.Password)
	return &Mailer{
		Dialer: d,
		ThisSender: cfg.ThisSender,
	}
}

func (m *Mailer) SendLoginLink(email, token string) error {
	msg := gomail.NewMessage()
	msg.SetHeader("From", m.ThisSender)
	msg.SetHeader("To", email)
	msg.SetHeader("Subject", "Login to organizer")

	buf := &bytes.Buffer{}
	tmplLoginLink.Execute(buf, token)

	msg.SetBody("text/plain", buf.String())

	err := m.Dialer.DialAndSend(msg)
	if err != nil {
		return err
	}
	return nil
}

const loginLinkBody = `
Login requested

Somebody has requested to login using your email.
If that wasn't you, you can ignore this email.

Use the following link {{.}} to sign in.

This link is single-use only and will expire after 10 minutes.
`
