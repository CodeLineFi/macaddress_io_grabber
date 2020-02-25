package mail

import (
	"log"
	"macaddress_io_grabber/config"
	"net/smtp"
	"strings"
)

func SendMail(config *config.Config, message string) {

	var (
		host     = config.AutoUpdate.Email.Host
		username = config.AutoUpdate.Email.Username
		password = config.AutoUpdate.Email.Password
		mailFrom = config.AutoUpdate.Email.EmailFrom
		mailTo   = config.AutoUpdate.Email.EmailTo
	)

	msg := []byte("To: " + mailTo[0] + "\r\n" +
		"Subject: MAC Grabber - Self-monitoring report\r\n" +
		"\r\n" +
		"The self-monitoring system detected a problem with the data collection or processing.\n\n" +
		"Thus today's data feeds are not deployed. Please review logs and the system's state to find out the reason and fix it.\n\n" +
		"Details: " + message + "\r\n")

	err := smtp.SendMail(host, smtp.PlainAuth("", username, password, strings.SplitN(host, ":", 2)[0]), mailFrom, mailTo, msg)
	if err != nil {
		log.Println(err)
		log.Println(message)
	}
}
