package mailer

import (
	"crypto/tls"
	"fmt"
	"net/smtp"
)

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

	return m.sendOne(toEmail, subject, body)
}

func (m *Mailer) SendRelease(toEmail, unsubscribeToken string, releaseTag, releaseName, releaseURL string) error {
	subject := fmt.Sprintf("New Release for %s: %s", releaseName, releaseTag)

	body, err := renderReleaseEmail(
		releaseName,
		releaseTag,
		releaseURL,
		fmt.Sprintf("%s/unsubscribe/%s", m.appBaseURL, unsubscribeToken),
	)
	if err != nil {
		return err
	}

	return m.sendOne(toEmail, subject, body)
}

func (m *Mailer) sendOne(toEmail, subject, body string) error {
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

	msg := fmt.Sprintf(
		"From: %s\r\nTo: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: text/html; charset=\"UTF-8\"\r\n\r\n%s",
		m.fromEmail, toEmail, subject, body,
	)

	if err := client.Mail(m.fromEmail); err != nil {
		return fmt.Errorf("MAIL FROM: %w", err)
	}

	if err := client.Rcpt(toEmail); err != nil {
		_ = client.Reset()
		return fmt.Errorf("RCPT TO: %w", err)
	}

	w, err := client.Data()
	if err != nil {
		_ = client.Reset()
		return fmt.Errorf("DATA: %w", err)
	}

	if _, err := w.Write([]byte(msg)); err != nil {
		_ = w.Close()
		_ = client.Reset()
		return fmt.Errorf("write message: %w", err)
	}

	return w.Close()
}
