// config/smtp.go
package config

import (
	"io"
	"log"
	"strconv"

	"gopkg.in/gomail.v2"
)

// SendEmail sends an email to the given recipients with the given subject and body.
// The email is sent via SMTP using the server, username, and password configured in the AppConfig.
func SendEmail(recipients []string, subject, body string) error {
	log.Printf("Sending email to: %v", recipients)
	m := gomail.NewMessage()
	// Set the "From" field with the desired format: "Theransotics.com"
	// The email address is the same as the SmtpEmail address.
	fromEmail := AppConfig.SmtpFromEmail
	fromName := "Theransotics.com"
	m.SetHeader("From", fromName+" <"+fromEmail+">")
	m.SetHeader("To", recipients...)
	m.SetHeader("Subject", subject)
	// Set the body to be HTML content.
	m.SetBody("text/html", body)

	// Create a new dialer with the SMTP server details.
	// Use configurable port, default to 587 for Mailjet
	port := 587
	if AppConfig.SmtpPort != "" {
		if p, err := strconv.Atoi(AppConfig.SmtpPort); err == nil {
			port = p
		}
	}
	d := gomail.NewDialer(AppConfig.SmtpServer, port, AppConfig.SmtpEmail, AppConfig.SmtpPassword)
	
	// Enable TLS for Mailjet
	d.TLSConfig = nil // Let gomail handle TLS automatically

	// Dial the SMTP server and send the email.
	if err := d.DialAndSend(m); err != nil {
		// Log the error if the email can't be sent.
		log.Printf("Failed to send email: %v", err)
		return err
	}

	// Log a success message if the email is sent successfully.
	log.Println("Email sent successfully")
	return nil
}

// SendEmailWithAttachment sends an email with a PDF attachment to the given recipients.
func SendEmailWithAttachment(recipients []string, subject, body, attachmentName string, attachmentBytes []byte) error {
	log.Printf("Sending email with attachment to: %v", recipients)
	m := gomail.NewMessage()
	fromEmail := AppConfig.SmtpFromEmail
	fromName := "Theransotics.com"
	m.SetHeader("From", fromName+" <"+fromEmail+">")
	m.SetHeader("To", recipients...)
	m.SetHeader("Subject", subject)
	m.SetBody("text/html", body)
	// Attach the PDF
	m.Attach(attachmentName, gomail.SetCopyFunc(func(w io.Writer) error {
		_, err := w.Write(attachmentBytes)
		return err
	}))

	// Use configurable port, default to 587 for Mailjet
	port := 587
	if AppConfig.SmtpPort != "" {
		if p, err := strconv.Atoi(AppConfig.SmtpPort); err == nil {
			port = p
		}
	}
	d := gomail.NewDialer(AppConfig.SmtpServer, port, AppConfig.SmtpEmail, AppConfig.SmtpPassword)
	
	// Enable TLS for Mailjet
	d.TLSConfig = nil // Let gomail handle TLS automatically
	
	if err := d.DialAndSend(m); err != nil {
		log.Printf("Failed to send email with attachment: %v", err)
		return err
	}
	log.Println("Email with attachment sent successfully")
	return nil
}
