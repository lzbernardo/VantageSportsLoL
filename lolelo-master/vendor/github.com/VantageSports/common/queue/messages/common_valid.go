package messages

import (
	"errors"
	"net/mail"
)

func (e *Email) Valid() error {
	// The only required fields for the sendgrid API are subject,
	// from (from_addr), and to (emails)
	if len(e.Emails) < 1 {
		return errors.New("at least one value for 'emails' is required")
	}
	for i := range e.Emails {
		if _, err := mail.ParseAddress(e.Emails[i]); err != nil {
			return err
		}
	}
	if e.Subject == "" || e.FromAddr == "" || (e.Body == "" && e.HtmlBody == "") {
		return errors.New("subject, from_addr, and one of body/html_body is required")
	}
	return nil
}
