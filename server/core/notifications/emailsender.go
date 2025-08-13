package notifications

type EmailSender interface {
	// SendEmail sends an email with the specified subject and body to the given recipient.
	SendEmail(to, subject, body string) error
}
