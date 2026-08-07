// emails/error_notification_email.go

package emails

import (
	"fmt"
	"theransticslabs/m/models"
)

// ErrorNotificationBody constructs the HTML body for the error notification email.
func ErrorNotificationEmail(errorLogs []models.ErrorLog) string {
	bodyContent := "<table width='100%' cellspacing='0' cellpadding='0'>" +
		"<tbody>" +
		"<tr><td height='30'></td></tr>" +
		"<tr><td style='color: #000; font-weight: 700; text-align: center;'>Error Notification</td></tr>" +
		"<tr><td height='20'></td></tr>" +
		"<tr><td style=''>The following errors have been detected in the system:</td></tr>" +
		"<tr><td height='20'></td></tr>" +
		"<tr><td><ul style='padding-left: 20px;'>"

	// Loop through each error log and add to the email body
	for _, log := range errorLogs {
		bodyContent += fmt.Sprintf(
			"<li style='margin-bottom: 15px;'>"+
				"<strong>API Path:</strong> %s<br>"+
				"<strong>Error:</strong> %s<br>"+
				"<strong>Type:</strong> %s</li>",
			log.APIPath, log.ErrorMessage, log.ErrorType)
	}

	bodyContent += "</ul></td></tr>" +
		"<tr><td height='20'></td></tr>" +
		"<tr><td style='text-align: center;'>Please check your application logs for further details.</td></tr>" +
		"<tr><td height='20'></td></tr>" +
		"<tr><td style='text-align: center;'>If you have any questions, please contact us at " +
		"<a href='mailto:official.labs@theranostic.com' style='color:#75AC71; text-decoration: underline;'>official.labs@theranostic.com</a>.</td></tr>" +
		"<tr><td height='30'></td></tr>" +
		"</tbody>" +
		"</table>"

	return CommonEmailTemplate(bodyContent)
}
