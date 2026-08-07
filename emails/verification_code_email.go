package emails

import (
	"fmt"
	"theransticslabs/m/utils"
)

func VerificationCodeEmail(
	firstName,
	lastName,
	verificationCode string,
) string {

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
      "
    >
      E-Consent Verification Code
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
      A verification code has been generated to confirm your e-consent submission.
      Please enter the code below to continue.
    </p>

    <div
      style="
        margin: 24px 0;
        text-align: center;
      "
    >
      <span
        style="
          display: inline-block;
          padding: 16px 32px;
          font-size: 32px;
          font-weight: 700;
          letter-spacing: 8px;
          color: #75ac71;
          border: 1px solid #d9d9d9;
          border-radius: 8px;
          background-color: #fafafa;
        "
      >
        %s
      </span>
    </div>

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
      <strong>Important:</strong> This verification code will expire in 5 minutes
      and can only be used once.
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
      For your security, please do not share this verification code with anyone.
    </p>
  </td>
</tr>

<tr>
  <td>
    <table style="width: 100%%" border="0" cellpadding="0" cellspacing="0">
      <tr>
        <td style="padding: 0 30px">
`,
		utils.CapitalizeWords(fmt.Sprintf("%s %s", firstName, lastName)),
		verificationCode,
	)

	return CommonEmailTemplate(bodyContent)
}
