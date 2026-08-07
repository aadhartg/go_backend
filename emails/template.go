package emails

import (
	"fmt"
	"theransticslabs/m/config"
)

func CommonEmailTemplate(bodyContent string) string {
	apiUrl := config.AppConfig.ApiUrl
	appUrl := config.AppConfig.AppUrl

	return fmt.Sprintf(`
<!DOCTYPE html>
<html>
  <head>
    <meta charset="utf-8" />
    <meta name="viewport" content="width=device-width, initial-scale=1" />
    <title>Theranostics</title>
    <link rel="preconnect" href="https://fonts.googleapis.com" />
    <link rel="preconnect" href="https://fonts.gstatic.com" crossorigin />
    <link
      href="https://fonts.googleapis.com/css2?family=Inria+Sans:ital,wght@0,300;0,400;0,700;1,300;1,400;1,700&display=swap"
      rel="stylesheet"
    />
  </head>
  <body>
    <table class="email-container" cellspacing="0" cellpadding="0" width='640' align="center" style="border: 1px solid #E2E2E2; border-radius: 16px; font-family: Arial, sans-serif;">
      <tr>
        <td>
          <table style="background: #75ac71; width: 100%%; border-top-left-radius: 16px; border-top-right-radius: 16px;">
            <tr height="10"></tr>
            <tr>
              <td width="20"></td>
              <td class="email-header">
                <a href="%s">
                  <img
                    src="%s/images/logo.png"
                    alt="Logo"
                    style="max-width: 100%%; height: auto;"
                  />
                </a>
              </td>
              <td align="right">
                <a href="%s">
                  <img
                    src="%s/images/vector.png"
                    alt="Vector"
                    style="max-width: 100%%; height: auto;"
                  />
                </a>
              </td>
              <td width="20"></td>
            </tr>
            <tr height="10"></tr>
          </table>
        </td>
      </tr>
      %s
        <p
                  style="
                    line-height: 20px;
                    font-size: 14px;
                    font-weight: 400;
                    color: #545454;
                    margin-bottom: 50px;
                    margin-top: 0px;
                  "
                >
                  Thank you for choosing Theranostics! We're here to ensure your
                  experience remains seamless and secure.
                </p>
              </td>
            </tr>
          </table>
        </td>
      </tr>
      <tr>
        <td style="text-align: center; padding: 15px; font-size: 14px; color: #777777;">
          <p style="margin-top: 0px; margin-bottom: 5px">
            If you have any questions, please email us at
          </p>
          <p style="margin-top: 0px; margin-bottom: 40px">
            <a style="color: #75ac71; text-decoration: underline; font-weight: 700;"
              href="mailto:info@theranosticslab.com">
              info@theranosticslab.com</a>
            or <a href="https://www.theranostics.co.nz/faqs/" style="color: #75ac71; text-decoration: underline;">
              Visit our FAQ page</a> on our website.
          </p>
        </td>
      </tr>
    </table>
  </body>
</html>`, appUrl, apiUrl, appUrl, apiUrl, bodyContent)
}

// PatientResignRequestEmail returns the HTML body for a patient re-signing request notification.
func PatientResignRequestEmail(patientName, eConsentTitle, portalLink string) string {
	body := `<tr><td style="padding: 30px"><p>Dear ` + patientName + `,</p>` +
		`<p>This is to inform you that you are required to re-sign the e-consent form for: <strong>` + eConsentTitle + `</strong>.</p>` +
		`<p>Please log in to your patient portal to review and re-sign the form.</p>` +
		`<p><a href="` + portalLink + `" style="background:#007bff;color:#fff;padding:10px 20px;text-decoration:none;border-radius:4px;">Go to Patient Portal</a></p>` +
		`<p>If you have any questions, please contact our support team.</p>` +
		`<p>Thank you,<br/>Theranostics Team</p> 	</td>
 <tr><td>
          <table style="width: 100%%" border="0" cellpadding="0" cellspacing="0">
            <tr>
              <td style="padding: 0 30px">`
	return CommonEmailTemplate(body)
}


// CoordinatorConsentWithdrawnEmail generates an email for coordinators when a patient withdraws consent
func CoordinatorConsentWithdrawnEmail(patientName, eConsentTitle, researchTitle, reason string) string {
	bodyContent := fmt.Sprintf(`
	<tr>
		<td style="padding: 30px">
			<h2 style="margin-top: 0; margin-bottom: 16px; color: #000; font-size: 26px; line-height: 32px; font-weight: 700; margin-bottom: 8px;">
				Patient Consent Withdrawal Notification
			</h2>
			<p style="line-height: 20px; font-size: 14px; font-weight: 400; color: #545454; margin-bottom: 8px; margin-top: 0px;">
				Dear Coordinator,
			</p>
			<p style="line-height: 20px; font-size: 14px; font-weight: 400; color: #545454; margin-bottom: 8px; margin-top: 0px;">
				Patient <strong>%s</strong> has withdrawn their consent for the e-consent form <strong>'%s'</strong> in research <strong>'%s'</strong> and all previous approved versions.
			</p>
			<p style="line-height: 20px; font-size: 14px; font-weight: 400; color: #545454; margin-bottom: 8px; margin-top: 0px;">
				<strong>Reason for withdrawal:</strong> %s
			</p>
		</td>
	</tr>
	<td>
          <table style="width: 100%%" border="0" cellpadding="0" cellspacing="0">
            <tr>
              <td style="padding: 0 30px">
	
	`, patientName, eConsentTitle, researchTitle, reason)
	return CommonEmailTemplate(bodyContent)
}

// PatientConsentWithdrawnReviewedEmail generates an email for patients when their withdrawn e-consent form has been reviewed.
func PatientConsentWithdrawnReviewedEmail(patientName string) string {
	bodyContent := `<tr><td style="padding: 30px"><p>Dear ` + patientName + `,</p>` +
		`<p>Your withdrawn e-consent form has been reviewed by our team.</p>` +
		`<p>If you have any questions or require further information, please contact our support team.</p>` +
		`<p>Thank you,<br/>Theranostics Team</p></td></tr>`
	return CommonEmailTemplate(bodyContent)
}
