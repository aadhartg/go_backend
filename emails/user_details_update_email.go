// emails/user_details_update_email.go
package emails

import (
	"fmt"
	"theransticslabs/m/utils"
)

// UserDetailsUpdatedEmail constructs the HTML body for the UserDetailsUpdatedEmail.
func UserDetailsUpdatedEmail(firstName, lastName, email, appUrl string) string {
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
      Your Account Details have been updated
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
      We wanted to let you know that your account details have been successfully updated. Here are your updated details for reference:
      </p>
      <table>
        <tr>
          <td>
            <table style="width: 100%%" border="0" cellpadding="0" cellspacing="0" >
              <tr>
                <td>
                  <p
                    style="
                      margin-bottom: 8px;
                      margin-top: 0px;
                      color: #545454;
                      font-size: 14px;
                    "
                  >
                    <strong
                      style="color: #000; font-size: 14px; font-weight: 600"
                      >Full Name:</strong
                    >
                    %s
                  </p>
                </td>
              </tr>
              <tr>
                <td>
                  <p
                    style="
                      margin-bottom: 8px;
                      margin-top: 0px;
                      color: #545454;
                      font-size: 14px;
                    "
                  >
                    <strong
                      style="color: #000; font-size: 14px; font-weight: 600"
                      >Email:</strong
                    >
                   %s
                  </p>
                </td>
              </tr>
             
            </table>
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
	`, utils.CapitalizeWords(fmt.Sprintf("%s %s", firstName, lastName)), utils.CapitalizeWords(fmt.Sprintf("%s %s", firstName, lastName)), email)

	return CommonEmailTemplate(bodyContent)
}
