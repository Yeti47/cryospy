package notifications

import (
	"crypto/tls"
	"fmt"
	"net/smtp"
)

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
	addr := fmt.Sprintf("%s:%d", s.Host, s.Port)

	// Create proper MIME message
	msg := s.buildMessage(to, subject, body)

	// Handle TLS connection for common secure ports
	if s.Port == 465 || s.Port == 587 {
		return s.sendWithTLS(addr, to, msg)
	}

	// Fallback to plain SMTP
	auth := smtp.PlainAuth("", s.Username, s.Password, s.Host)
	return smtp.SendMail(addr, auth, s.From, []string{to}, msg)
}

func (s *SmtpSender) buildMessage(to, subject, body string) []byte {
	headers := make(map[string]string)
	headers["From"] = s.From
	headers["To"] = to
	headers["Subject"] = subject
	headers["MIME-Version"] = "1.0"
	headers["Content-Type"] = "text/plain; charset=\"utf-8\""
	headers["Content-Transfer-Encoding"] = "7bit"

	message := ""
	for k, v := range headers {
		message += fmt.Sprintf("%s: %s\r\n", k, v)
	}
	message += "\r\n" + body

	return []byte(message)
}

func (s *SmtpSender) sendWithTLS(addr, to string, msg []byte) error {
	// Create TLS connection
	tlsConfig := &tls.Config{
		InsecureSkipVerify: false,
		ServerName:         s.Host,
	}

	conn, err := tls.Dial("tcp", addr, tlsConfig)
	if err != nil {
		return fmt.Errorf("failed to connect with TLS: %w", err)
	}
	defer conn.Close()

	client, err := smtp.NewClient(conn, s.Host)
	if err != nil {
		return fmt.Errorf("failed to create SMTP client: %w", err)
	}
	defer client.Quit()

	// Authenticate
	auth := smtp.PlainAuth("", s.Username, s.Password, s.Host)
	if err = client.Auth(auth); err != nil {
		return fmt.Errorf("authentication failed: %w", err)
	}

	// Set sender
	if err = client.Mail(s.From); err != nil {
		return fmt.Errorf("failed to set sender: %w", err)
	}

	// Set recipient
	if err = client.Rcpt(to); err != nil {
		return fmt.Errorf("failed to set recipient: %w", err)
	}

	// Send message
	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("failed to get data writer: %w", err)
	}

	_, err = w.Write(msg)
	if err != nil {
		return fmt.Errorf("failed to write message: %w", err)
	}

	return w.Close()
}
