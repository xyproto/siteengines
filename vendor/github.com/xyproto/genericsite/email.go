package genericsite

import (
	"net/smtp"
)

// TODO: Forgot password email
// TODO: Forgot username email
// TODO: "click here if you have not asked for this"

func ConfirmationEmail(domain, link, username, email string) error {
	host := "localhost"
	auth := smtp.PlainAuth("", "", "", host)
	msgString := "From: " + domain + " <noreply@" + domain + ">\n"
	msgString += "To: " + email + "\n"
	msgString += "Subject: Welcome, " + username + "\n"
	msgString += "\n"
	msgString += "Hi and welcome to " + domain + "!\n"
	msgString += "\n"
	msgString += "Confirm the registration by following this link:\n"
	msgString += link + "\n"
	msgString += "\n"
	msgString += "Thank you.\n"
	msgString += "\n"
	msgString += "Best regards,\n"
	msgString += "    The " + domain + " registration system\n"
	msg := []byte(msgString)
	from := "noreply@" + domain
	to := []string{email}
	hostPort := host + ":25"
	return smtp.SendMail(hostPort, auth, from, to, msg)
}
