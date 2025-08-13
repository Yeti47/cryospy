package notifications

type nopSender struct{}

var NopSender EmailSender = &nopSender{}

// SendEmail does nothing and returns nil.
func (n *nopSender) SendEmail(to, subject, body string) error {
	return nil
}
