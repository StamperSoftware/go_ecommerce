package main

import (
	"bytes"
	"embed"
	"fmt"
	mail "github.com/xhit/go-simple-mail/v2"
	"html/template"
	"time"
)

//go:embed templates
var emailTemplatesFS embed.FS

func (app *application) SendMail(from, to, subject, tmpl string, data interface{}) error {
	templateToRender := fmt.Sprintf("templates/%s.html.gohtml", tmpl)

	t, err := template.New("email-html").ParseFS(emailTemplatesFS, templateToRender)

	if err != nil {
		app.errorLog.Println(err)
		return err
	}

	var tpl bytes.Buffer
	err = t.ExecuteTemplate(&tpl, "body", data)

	if err != nil {
		app.errorLog.Println(err)
		return err
	}

	formattedMessage := tpl.String()

	templateToRender = fmt.Sprintf("templates/%s.plain.gohtml", tmpl)

	t, err = template.New("email-plain").ParseFS(emailTemplatesFS, templateToRender)

	if err != nil {
		app.errorLog.Println(err)
		return err
	}

	err = t.ExecuteTemplate(&tpl, "body", data)

	if err != nil {
		app.errorLog.Println(err)
		return err
	}

	plainMessage := tpl.String()

	server := mail.NewSMTPClient()
	server.Host = app.config.smtp.host
	server.Password = app.config.smtp.password
	server.Username = app.config.smtp.username
	server.Port = app.config.smtp.port
	server.Encryption = mail.EncryptionTLS
	server.KeepAlive = false
	server.ConnectTimeout = 10 * time.Second
	server.SendTimeout = 10 * time.Second

	smtpClient, err := server.Connect()

	if err != nil {
		return err
	}

	email := mail.NewMSG()
	email.SetFrom(from).AddTo(to).SetSubject(subject)
	email.SetBody(mail.TextHTML, formattedMessage)
	email.AddAlternative(mail.TextPlain, plainMessage)

	err = email.Send(smtpClient)

	if err != nil {
		app.errorLog.Println(err)
		return err
	}

	app.infoLog.Println("Sent mail")

	return nil
}
