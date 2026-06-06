package notification

import (
	"crypto/tls"
	"errors"
	"fmt"
	"net/smtp"
)

type ReleaseRecipient struct {
	Email            string
	UnsubscribeToken string
}

type ReleaseInfo struct {
	Tag  string
	Name string
	URL  string
}

type Mailer struct {
	host       string
	port       string
	user       string
	pass       string
	fromEmail  string
	appBaseURL string
}

func NewMailer(host, port, user, pass, fromEmail, appBaseURL string) *Mailer {
	return &Mailer{
		host:       host,
		port:       port,
		user:       user,
		pass:       pass,
		fromEmail:  fromEmail,
		appBaseURL: appBaseURL,
	}
}

func (m *Mailer) SendConfirmation(toEmail, repoName, confirmToken string) error {
	subject := fmt.Sprintf("Confirm subscription: %s", repoName)

	body, err := renderConfirmationEmail(repoName, fmt.Sprintf("%s/confirm/%s", m.appBaseURL, confirmToken))
	if err != nil {
		return err
	}

	return m.send(toEmail, subject, body)
}

func (m *Mailer) SendReleaseEmails(recipients []ReleaseRecipient, release ReleaseInfo) error {
	if len(recipients) == 0 {
		return nil
	}

	address := fmt.Sprintf("%s:%s", m.host, m.port)

	client, err := smtp.Dial(address)
	if err != nil {
		return fmt.Errorf("dial smtp: %w", err)
	}
	defer func() { _ = client.Close() }()

	if ok, _ := client.Extension("STARTTLS"); ok {
		if err := client.StartTLS(&tls.Config{ServerName: m.host}); err != nil {
			return fmt.Errorf("smtp starttls: %w", err)
		}
	}

	if m.user != "" && m.pass != "" {
		auth := smtp.PlainAuth("", m.user, m.pass, m.host)
		if err := client.Auth(auth); err != nil {
			return fmt.Errorf("smtp auth: %w", err)
		}
	}

	var errs []error

	for _, r := range recipients {
		if sendErr := m.sendOneViaClient(client, r, release); sendErr != nil {
			errs = append(errs, fmt.Errorf("notify %s: %w", r.Email, sendErr))

			if err := client.Reset(); err != nil {
				errs = append(errs, fmt.Errorf("smtp reset failed, aborting remaining sends: %w", err))

				break
			}
		}
	}

	return errors.Join(errs...)
}

func (m *Mailer) sendOneViaClient(client *smtp.Client, r ReleaseRecipient, release ReleaseInfo) error {
	subject := fmt.Sprintf("New Release for %s: %s", release.Name, release.Tag)

	body, err := renderReleaseEmail(
		release.Name,
		release.Tag,
		release.URL,
		fmt.Sprintf("%s/unsubscribe/%s", m.appBaseURL, r.UnsubscribeToken),
	)
	if err != nil {
		return err
	}

	msg := fmt.Sprintf(
		"From: %s\r\nTo: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: text/html; charset=\"UTF-8\"\r\n\r\n%s",
		m.fromEmail, r.Email, subject, body,
	)

	if err := client.Mail(m.fromEmail); err != nil {
		return fmt.Errorf("MAIL FROM: %w", err)
	}

	if err := client.Rcpt(r.Email); err != nil {
		return fmt.Errorf("RCPT TO: %w", err)
	}

	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("DATA: %w", err)
	}

	if _, err := w.Write([]byte(msg)); err != nil {
		_ = w.Close()

		return fmt.Errorf("write message: %w", err)
	}

	return w.Close()
}

func (m *Mailer) send(toEmail, subject, body string) error {
	var auth smtp.Auth
	if m.user != "" && m.pass != "" {
		auth = smtp.PlainAuth("", m.user, m.pass, m.host)
	}

	msg := fmt.Sprintf(
		"From: %s\r\nTo: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: text/html; charset=\"UTF-8\"\r\n\r\n%s",
		m.fromEmail, toEmail, subject, body,
	)

	address := fmt.Sprintf("%s:%s", m.host, m.port)
	if err := smtp.SendMail(address, auth, m.fromEmail, []string{toEmail}, []byte(msg)); err != nil {
		return fmt.Errorf("send email: %w", err)
	}

	return nil
}
