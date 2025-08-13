package notifications

import (
	"fmt"
	"net/smtp"
)

var sendMail = smtp.SendMail

// SmtpSender implements the EmailSender interface using SMTP.
type SmtpSender struct {
	Host     string
	Port     int
	Username string
	Password string
	From     string
}

// NewSmtpSender creates a new SmtpSender.
func NewSmtpSender(host string, port int, username, password, from string) *SmtpSender {
	return &SmtpSender{
		Host:     host,
		Port:     port,
		Username: username,
		Password: password,
		From:     from,
	}
}

// SendEmail sends an email using SMTP.
func (s *SmtpSender) SendEmail(to, subject, body string) error {
	auth := smtp.PlainAuth("", s.Username, s.Password, s.Host)

	msg := []byte("To: " + to + "\r\n" +
		"From: " + s.From + "\r\n" +
		"Subject: " + subject + "\r\n" +
		"\r\n" +
		body + "\r\n")

	addr := fmt.Sprintf("%s:%d", s.Host, s.Port)

	return sendMail(addr, auth, s.From, []string{to}, msg)
}
