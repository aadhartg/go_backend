package emails

import (
	"fmt"
	"theransticslabs/m/utils"
)

// GenomicResultPublishedEmail generates an email template for genomic result publication notification
func GenomicResultPublishedEmail(
	patientFirstName string,
	patientLastName string,
	researchTitle string,
	econsentTitle string,
	resultLink string,
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
              margin-bottom: 8px;
            "
          >
            Your Genomic Results Are Ready
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
            We are pleased to inform you that your genomic results for the research study <strong>'%s'</strong> and e-consent form <strong>'%s'</strong> have been published and are now available for your review.
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
            You can now access and download your results from our secure portal. Please log in to your patient account to view the complete report.
          </p>
          <table>
            <tr>
              <td>
                <table style="width: 100%%" border="0" cellpadding="0" cellspacing="0" >
                  <tr>
                    <td>
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
                        <strong>Important:</strong> Your genomic results contain sensitive medical information. Please ensure you access them from a secure, private location and do not share them with unauthorized individuals.
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
                  >View Results</a
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
                  If you have any questions about your results or need assistance understanding the report, please don't hesitate to contact our support team.
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
                  Thank you for participating in our research study.
                </p>
              </td>
            </tr>
          </table>
        </td>
      </tr>
			<td>
          <table style="width: 100%%" border="0" cellpadding="0" cellspacing="0">
            <tr>
              <td style="padding: 0 30px">
	`, 
		utils.CapitalizeWords(fmt.Sprintf("%s %s", patientFirstName, patientLastName)),
		researchTitle,
		econsentTitle,
		resultLink)

	return CommonEmailTemplate(bodyContent)
} 