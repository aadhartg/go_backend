// emails/user_status_update_change_email.go
package emails

import (
	"fmt"
	"theransticslabs/m/utils"
)

// UserStatusChangedEmail constructs the HTML body for the account status email.
// 'status' should be "activated" or "deactivated" to dynamically adjust the content.
func UserStatusChangedEmail(firstName, lastName, status, appUrl string) string {
	var subject, message, buttonText, buttonUrl string

	// Determine content based on status
	if status == "activated" {
		subject = "Your Account has been Activated"
		message = "We are pleased to inform you that your account has been successfully <strong>activated</strong>. You now have full access to our platform and its features."
		buttonText = "Login to Your Account"
		buttonUrl = appUrl + "/login"
	} else if status == "deactivated" {
		subject = "Your Account has been Deactivated"
		message = "We want to inform you that your account has been <strong>deactivated</strong>. This means you will no longer have access to your account and its associated features."
		buttonText = "Contact Support"
		buttonUrl = "https://www.theranostics.co.nz/contact/"
	}

	bodyContent := fmt.Sprintf(`
	<tr>
        <td style="padding: 30px">
          <h2
            style="
              margin-top: 0;
              margin-bottom: 16px;
              color: #000;
              font-size: 26px;
              line-height: 32px;
              font-weight: 700;
              margin-bottom: 8px;
            "
          >
            %s
          </h2>
          <p
            style="
              line-height: 20px;
              font-size: 14px;
              font-weight: 400;
              color: #545454;
              margin-bottom: 8px;
              margin-top: 0px;
            "
          >
            Dear %s,
          </p>
          <p
            style="
              line-height: 20px;
              font-size: 14px;
              font-weight: 400;
              color: #545454;
              margin-bottom: 8px;
              margin-top: 0px;
            "
          >
           %s
          </p>
        
        </td>
      </tr>
      <tr>
        <td>
          <table style="width: 100%%" border="0" cellpadding="0" cellspacing="0">
            <tr>
              <td align="center">
                <a
                  href="%s"
                  style="
                    margin:0 0 30px 0;
                    font-weight: 700;
                    font-size: 14px;
                    color: #fff;
                    background: #75ac71;
                    height: 38px;
                    width: 190px;
                    display: inline-block;
                    text-align: center;
                    line-height: 38px;
                    border-radius: 4px;
                    text-decoration: none;
                  "
                  >%s</a
                >
              </td>
            </tr>
          </table>
        </td>
      </tr>
      <tr>
        <td>
          <table style="width: 100%%" border="0" cellpadding="0" cellspacing="0">
            <tr>
              <td style="padding: 0 30px">
               
	`, subject, utils.CapitalizeWords(fmt.Sprintf("%s %s", firstName, lastName)), message, buttonUrl, buttonText)

	return CommonEmailTemplate(bodyContent)
}
