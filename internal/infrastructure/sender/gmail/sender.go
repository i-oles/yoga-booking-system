package gmail

import (
	"crypto/tls"
	"fmt"

	"main/internal/infrastructure/sender"

	"gopkg.in/gomail.v2"
)

type Sender struct {
	dialer *gomail.Dialer
}

func NewSender(
	host string,
	port int,
	login string,
	password string,
) *Sender {
	dialer := gomail.NewDialer(host, port, login, password)
	dialer.TLSConfig = &tls.Config{
		MinVersion:         tls.VersionTLS12,
		ServerName:         host,
		InsecureSkipVerify: false,
	}

	return &Sender{
		dialer: dialer,
	}
}

func (s *Sender) Send(messages ...sender.Message) error {
	msgs := make([]*gomail.Message, len(messages))
	for i, m := range messages {
		msg := gomail.NewMessage()
		msg.SetHeader("From", m.From)
		msg.SetHeader("To", m.To)
		msg.SetHeader("Subject", m.Subject)
		msg.SetBody("text/html", m.Body)
		msgs[i] = msg
	}

	if err := s.dialer.DialAndSend(msgs...); err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}

	return nil
}
