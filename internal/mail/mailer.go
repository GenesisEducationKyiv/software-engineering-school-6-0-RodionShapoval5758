package mail

import (
	"bytes"
	"crypto/tls"
	"errors"
	"fmt"
	"html/template"
	"net/smtp"

	"GithubReleaseNotificationAPI/internal/domain"
)

var confirmationTmpl = template.Must(template.New("confirmation").Parse(`
<p>Confirm subscription to <b>{{.RepoName}}</b>:</p>
<p>
	<a href="{{.ConfirmLink}}" style="background-color: #2ea44f; color: white; padding: 10px 20px; text-decoration: none; border-radius: 5px; display: inline-block;">Confirm Subscription</a>
</p>
`))

var releaseTmpl = template.Must(template.New("release").Parse(`
<h3>New release available for <b>{{.ReleaseName}}</b></h3>
<p><b>Tag:</b> {{.Tag}}</p>
<p><b>Name:</b> {{.ReleaseName}}</p>
<p><a href="{{.ReleaseURL}}" style="background-color: #0366d6; color: white; padding: 8px 16px; text-decoration: none; border-radius: 5px; display: inline-block;">View Release on GitHub</a></p>
<p style="margin-top: 16px;">
	<a href="{{.UnsubscribeLink}}" style="color: #6a737d; text-decoration: underline;">Unsubscribe from these notifications</a>
</p>
`))

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

	var buf bytes.Buffer
	if err := confirmationTmpl.Execute(&buf, struct {
		RepoName    string
		ConfirmLink string
	}{
		RepoName:    repoName,
		ConfirmLink: fmt.Sprintf("%s/confirm/%s", s.appBaseURL, confirmToken),
	}); err != nil {
		return fmt.Errorf("render confirmation email: %w", err)
	}

	return s.send(toEmail, subject, buf.String())
}

// SendReleaseNotifications sends a release email to each subscriber over a single
// SMTP connection. Per-recipient failures are collected and returned via errors.Join.
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
			client.Reset() //nolint:errcheck
		}
	}

	return errors.Join(errs...)
}

func (s *SMTPService) sendOneViaClient(client *smtp.Client, sub domain.Subscription, release *domain.Release) error {
	subject := fmt.Sprintf("New Release for %s: %s", release.Name, release.Tag)

	var buf bytes.Buffer
	if err := releaseTmpl.Execute(&buf, struct {
		ReleaseName     string
		Tag             string
		ReleaseURL      string
		UnsubscribeLink string
	}{
		ReleaseName:     release.Name,
		Tag:             release.Tag,
		ReleaseURL:      release.URL,
		UnsubscribeLink: fmt.Sprintf("%s/unsubscribe/%s", s.appBaseURL, sub.UnsubscribeToken),
	}); err != nil {
		return fmt.Errorf("render release email: %w", err)
	}

	msg := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: text/html; charset=\"UTF-8\"\r\n\r\n%s",
		s.fromEmail, sub.Email, subject, buf.String())

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

	msg := fmt.Sprintf("From: %s\r\n"+
		"To: %s\r\n"+
		"Subject: %s\r\n"+
		"MIME-Version: 1.0\r\n"+
		"Content-Type: text/html; charset=\"UTF-8\"\r\n"+
		"\r\n"+
		"%s", s.fromEmail, toEmail, subject, body)

	address := fmt.Sprintf("%s:%s", s.host, s.port)
	err := smtp.SendMail(address, auth, s.fromEmail, []string{toEmail}, []byte(msg))
	if err != nil {
		return fmt.Errorf("send email: %w", err)
	}

	return nil
}
