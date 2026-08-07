package emails

import (
	"fmt"
	"theransticslabs/m/utils"
)

// ReportUploadNotificationEmail generates an email template for report upload notification
func ReportUploadNotificationEmail(
	customerFirstName string,
	customerLastName string,
	patientFirstName string,
	patientLastName string,
	reportLink string,
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
            Theranostics Lab Report
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
            The report for patient %s has been come.
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
                        You can now access and download the report from our secure portal.
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
                  >View Report</a
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
               
	`, utils.CapitalizeWords(fmt.Sprintf("%s %s", customerFirstName, customerLastName)),
		utils.CapitalizeWords(fmt.Sprintf("%s %s", patientFirstName, patientLastName)),
		reportLink)

	return CommonEmailTemplate(bodyContent)
}
