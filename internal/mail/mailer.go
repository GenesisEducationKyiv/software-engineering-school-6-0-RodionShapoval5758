package mail

import (
	"crypto/tls"
	"errors"
	"fmt"
	"net/smtp"

	"GithubReleaseNotificationAPI/internal/domain"
)

type SMTPService struct {
	host       string
	port       string
	user       string
	pass       string
	fromEmail  string
	appBaseURL string
}

func NewSMTPService(host, port, user, pass, fromEmail, appBaseURL string) *SMTPService {
	return &SMTPService{
		host:       host,
		port:       port,
		user:       user,
		pass:       pass,
		fromEmail:  fromEmail,
		appBaseURL: appBaseURL,
	}
}

func (s *SMTPService) SendConfirmationEmail(toEmail, repoName, confirmToken string) error {
	subject := fmt.Sprintf("Confirm subscription: %s", repoName)

	body, err := renderConfirmationEmail(repoName, fmt.Sprintf("%s/confirm/%s", s.appBaseURL, confirmToken))
	if err != nil {
		return err
	}

	return s.send(toEmail, subject, body)
}

func (s *SMTPService) SendReleaseNotifications(subscriptions []domain.Subscription, release *domain.Release) error {
	if len(subscriptions) == 0 {
		return nil
	}

	address := fmt.Sprintf("%s:%s", s.host, s.port)

	client, err := smtp.Dial(address)
	if err != nil {
		return fmt.Errorf("dial smtp: %w", err)
	}
	defer func() { _ = client.Close() }()

	if ok, _ := client.Extension("STARTTLS"); ok {
		if err := client.StartTLS(&tls.Config{ServerName: s.host}); err != nil {
			return fmt.Errorf("smtp starttls: %w", err)
		}
	}

	if s.user != "" && s.pass != "" {
		auth := smtp.PlainAuth("", s.user, s.pass, s.host)
		if err := client.Auth(auth); err != nil {
			return fmt.Errorf("smtp auth: %w", err)
		}
	}

	var errs []error

	for _, sub := range subscriptions {
		if sendErr := s.sendOneViaClient(client, sub, release); sendErr != nil {
			errs = append(errs, fmt.Errorf("notify %s: %w", sub.Email, sendErr))

			if err := client.Reset(); err != nil {
				errs = append(errs, fmt.Errorf("smtp reset failed, aborting remaining sends: %w", err))

				break
			}
		}
	}

	return errors.Join(errs...)
}

func (s *SMTPService) sendOneViaClient(client *smtp.Client, sub domain.Subscription, release *domain.Release) error {
	subject := fmt.Sprintf("New Release for %s: %s", release.Name, release.Tag)

	body, err := renderReleaseEmail(
		release.Name,
		release.Tag,
		release.URL,
		fmt.Sprintf("%s/unsubscribe/%s", s.appBaseURL, sub.UnsubscribeToken),
	)
	if err != nil {
		return err
	}

	msg := fmt.Sprintf(
		"From: %s\r\nTo: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: text/html; charset=\"UTF-8\"\r\n\r\n%s",
		s.fromEmail, sub.Email, subject, body,
	)

	if err := client.Mail(s.fromEmail); err != nil {
		return fmt.Errorf("MAIL FROM: %w", err)
	}

	if err := client.Rcpt(sub.Email); err != nil {
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

func (s *SMTPService) send(toEmail, subject, body string) error {
	var auth smtp.Auth
	if s.user != "" && s.pass != "" {
		auth = smtp.PlainAuth("", s.user, s.pass, s.host)
	}

	msg := fmt.Sprintf(
		"From: %s\r\nTo: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: text/html; charset=\"UTF-8\"\r\n\r\n%s",
		s.fromEmail, toEmail, subject, body,
	)

	address := fmt.Sprintf("%s:%s", s.host, s.port)
	if err := smtp.SendMail(address, auth, s.fromEmail, []string{toEmail}, []byte(msg)); err != nil {
		return fmt.Errorf("send email: %w", err)
	}

	return nil
}
