package controllers

import (
	"gopkg.in/gomail.v2"
	"log"
	"webroot/models"
)

func SendEmail(toEmail, Subject string, plainTextBody string) error {
	config := models.GetConfig()
	smtpConfig := config.Server.SMTP
	m := gomail.NewMessage()
	m.SetHeader("From", smtpConfig.Email)
	m.SetHeader("To", toEmail)
	m.SetHeader("Subject", Subject)
	m.SetBody("text/plain", plainTextBody)
	d := gomail.NewDialer(smtpConfig.Server, smtpConfig.Port, smtpConfig.Email, smtpConfig.Password)
	d.SSL = smtpConfig.SSL
	if err := d.DialAndSend(m); err != nil {
		log.Printf("发送邮件到 %s 失败: %v", toEmail, err)
		return err
	}
	log.Printf("成功发送邮件到 %s", toEmail)
	return nil
}
