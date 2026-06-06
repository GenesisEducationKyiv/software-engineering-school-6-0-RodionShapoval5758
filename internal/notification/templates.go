package notification

import (
	"bytes"
	"fmt"
	"html/template"
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

func renderConfirmationEmail(repoName, confirmLink string) (string, error) {
	var buf bytes.Buffer

	if err := confirmationTmpl.Execute(&buf, struct {
		RepoName    string
		ConfirmLink string
	}{
		RepoName:    repoName,
		ConfirmLink: confirmLink,
	}); err != nil {
		return "", fmt.Errorf("render confirmation email: %w", err)
	}

	return buf.String(), nil
}

func renderReleaseEmail(releaseName, tag, releaseURL, unsubscribeLink string) (string, error) {
	var buf bytes.Buffer

	if err := releaseTmpl.Execute(&buf, struct {
		ReleaseName     string
		Tag             string
		ReleaseURL      string
		UnsubscribeLink string
	}{
		ReleaseName:     releaseName,
		Tag:             tag,
		ReleaseURL:      releaseURL,
		UnsubscribeLink: unsubscribeLink,
	}); err != nil {
		return "", fmt.Errorf("render release email: %w", err)
	}

	return buf.String(), nil
}
