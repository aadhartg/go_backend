package emails

import (
	"fmt"
	"theransticslabs/m/config"
	"theransticslabs/m/utils"
)

func CustomerOrderConfirmationEmail(
	orderNumber,
	customerFirstName,
	customerLastName,
	customerEmail,
	customerPhoneNumber,
	customerStreetAddress,
	customerTownCity,
	customerRegion,
	customerPostcode,
	customerCountry,
	customerShippingStreetAddress,
	customerShippingTownCity,
	customerShippingRegion,
	customerShippingPostcode,
	customerShippingCountry,
	orderClinicID,
	productName,
	productDescription string,
	productQuantity int,
	productPrice,
	productGstPrice,
	productDiscount float64,
	orderTotalAmount float64,
	productImage,
	invoiceLink,
	appUrl string) string {
	apiUrl := config.AppConfig.ApiUrl

	// Calculate GST percentage
	gstPercentage := (productGstPrice / productPrice) * 100

	// Calculate discount value
	discountValue := (productPrice * productDiscount) / 100

	// Calculate subTotal
	subTotal := productPrice * float64(productQuantity)

	// Calculate totalGst
	gstTotal := productGstPrice * float64(productQuantity)

	// Calculate totalDiscount
	discountTotal := discountValue * float64(productQuantity)

	// Prepare clinic ID section
	clinicIDSection := ""
	if orderClinicID != "" {
		clinicIDSection = fmt.Sprintf(`
				<tr>
		  <td
			style="
			  padding-bottom: 10px;
			  font-size: 14px;
			  font-weight: 400;
			  color: #545454;
			  padding: 10px 0;
			  border-bottom: 1px solid #ebebeb;
			  white-space: nowrap;
			"
		  >
			Clinic ID:
		  </td>
		  <td
			align="right"
			style="
			  padding-bottom: 10px;
			  font-size: 14px;
			  font-weight: 400;
			  color: #000;
			  padding: 10px 0;
			  border-bottom: 1px solid #ebebeb;
			"
		  >
			%s
		  </td>
		</tr>`, orderClinicID)
	}

	bodyContent := fmt.Sprintf(`
	 <tr align="center">
        <td style="padding: 40px 20px 30px 20px">
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
            Order Confirmation – Order #%s
          </h2>
          <p
            style="
              line-height: 20px;
              font-size: 14px;
              font-weight: 400;
              color: #545454;
              margin-bottom: 0px;
              margin-top: 0px;
              padding: 0 70px;
            "
          >
           Thank you for your order with <strong>Theranostics</strong>! Below are your order details for reference:
          </p>
        </td>
      </tr>
      <tr>
        <td style="font-size: 18px; font-weight: 700; line-height: 20px">
          <table style="width: 100%%" cellspacing="0" cellpadding="0">
            <tr>
              <td
                style="
                  padding: 0 20px 0px 20px;
                  font-weight: 700;
                  font-size: 18px;
                  color: #000;
                "
              >
                Customer Infromation
              </td>
            </tr>
          </table>
        </td>
      </tr>
      <tr>
        <td>
          <table
            cellspacing="0"
            cellpadding="0"
            width="100%%"
            style="padding: 20px"
          >
            <tr>
              <td
                style="
                  padding-bottom: 10px;
                  font-size: 14px;
                  font-weight: 400;
                  color: #545454;
                  border-bottom: 1px solid #ebebeb;
                  margin-bottom: 10px;
                  white-space: nowrap;
                "
              >
                Name
              </td>
              <td
                align="right"
                style="
                  padding-bottom: 10px;
                  font-size: 14px;
                  font-weight: 400;
                  color: #000;
                  border-bottom: 1px solid #ebebeb;
                  margin-bottom: 10px;
                "
              >
                %s
              </td>
            </tr>
			 %s
            <tr>
              <td
                style="
                  padding-bottom: 10px;
                  font-size: 14px;
                  font-weight: 400;
                  color: #545454;
                  padding: 10px 0;
                  border-bottom: 1px solid #ebebeb;
                  white-space: nowrap;
                "
              >
                Email
              </td>
              <td
                align="right"
                style="
                  padding-bottom: 10px;
                  font-size: 14px;
                  font-weight: 400;
                  color: #000;
                  padding: 10px 0;
                  border-bottom: 1px solid #ebebeb;
                "
              >
                %s
              </td>
            </tr>
            <tr>
              <td
                style="
                  padding-bottom: 10px;
                  font-size: 14px;
                  font-weight: 400;
                  color: #545454;
                  padding: 10px 0;
                  border-bottom: 1px solid #ebebeb;
                  white-space: nowrap;
                "
              >
                Phone Number:
              </td>
              <td
                align="right"
                style="
                  padding-bottom: 10px;
                  font-size: 14px;
                  font-weight: 400;
                  color: #000;
                  padding: 10px 0;
                  border-bottom: 1px solid #ebebeb;
                "
              >
                %s
              </td>
            </tr>
            <tr>
              <td
                style="
                  padding-bottom: 10px;
                  font-size: 14px;
                  font-weight: 400;
                  color: #545454;
                  padding: 10px 0;
                  border-bottom: 1px solid #ebebeb;
                  white-space: nowrap;
                "
              >
                Billing Address:
              </td>
              <td
                align="right"
                style="
                  padding-bottom: 10px;
                  font-size: 14px;
                  font-weight: 400;
                  color: #000;
                  padding: 10px 0;
                  border-bottom: 1px solid #ebebeb;
                "
              >
                %s, %s, %s, %s, %s
              </td>
            </tr>
            <tr>
              <td
                style="
                  padding-bottom: 10px;
                  font-size: 14px;
                  font-weight: 400;
                  color: #545454;
                  padding: 10px 0;
                  white-space: nowrap;
                "
              >
                Shipping Address:
              </td>
              <td
                align="right"
                style="
                  padding-bottom: 10px;
                  font-size: 14px;
                  font-weight: 400;
                  color: #000;
                  padding: 10px 0;
                "
              >
               %s, %s, %s, %s, %s
              </td>
            </tr>
          </table>
        </td>
      </tr>

      <tr>
        <td style="font-size: 18px; font-weight: 700; line-height: 20px">
          <table style="width: 100%%" cellspacing="0" cellpadding="0">
            <tr>
              <td
                style="
                  padding: 0 20px 0px 20px;
                  font-weight: 700;
                  font-size: 18px;
                  color: #000;
                "
              >
                Order Information
              </td>
            </tr>
          </table>
        </td>
      </tr>
      <tr>
        <td>
          <table
            cellspacing="0"
            cellpadding="0"
            width="100%%"
            style="padding: 20px"
          >
            <tr>
              <td
                style="
                  padding-bottom: 10px;
                  font-size: 14px;
                  font-weight: 400;
                  color: #545454;
                  padding: 10px 0;
                  border-bottom: 1px solid #ebebeb;
                  white-space: nowrap;
                "
              >
                Order Number:
              </td>
              <td
                align="right"
                style="
                  padding-bottom: 10px;
                  font-size: 14px;
                  font-weight: 400;
                  color: #000;
                  padding: 10px 0;
                  border-bottom: 1px solid #ebebeb;
                "
              >
                %s
              </td>
            </tr>
            <tr>
              <td
                style="
                  padding-bottom: 10px;
                  font-size: 14px;
                  font-weight: 400;
                  color: #545454;
                  padding: 10px 0;
                  border-bottom: 1px solid #ebebeb;
                  white-space: nowrap;
                "
              >
                Product Name:
              </td>
              <td
                align="right"
                style="
                  padding-bottom: 10px;
                  font-size: 14px;
                  font-weight: 400;
                  color: #000;
                  padding: 10px 0;
                  border-bottom: 1px solid #ebebeb;
                "
              >
                %s
              </td>
            </tr>
            <tr>
              <td
                style="
                  padding-bottom: 10px;
                  font-size: 14px;
                  font-weight: 400;
                  color: #545454;
                  padding: 10px 0;
                  border-bottom: 1px solid #ebebeb;
                  white-space: nowrap;
                "
              >
                Product Description:
              </td>
              <td
                align="right"
                style="
                  padding-bottom: 10px;
                  font-size: 14px;
                  font-weight: 400;
                  color: #000;
                  padding: 10px 0;
                  border-bottom: 1px solid #ebebeb;
                  padding-left: 80px;
                "
              >
                %s
              </td>
            </tr>
            <tr>
              <td
                style="
                  padding-bottom: 10px;
                  font-size: 14px;
                  font-weight: 400;
                  color: #545454;
                  padding: 10px 0;
                  border-bottom: 1px solid #ebebeb;
                  white-space: nowrap;
                "
              >
                Quantity
              </td>
              <td
                align="right"
                style="
                  padding-bottom: 10px;
                  font-size: 14px;
                  font-weight: 400;
                  color: #000;
                  padding: 10px 0;
                  border-bottom: 1px solid #ebebeb;
                "
              >
                %d
              </td>
            </tr>
            <tr>
              <td
                style="
                  padding-bottom: 10px;
                  font-size: 14px;
                  font-weight: 400;
                  color: #545454;
                  padding: 10px 0;
                  border-bottom: 1px solid #ebebeb;
                  white-space: nowrap;
                "
              >
                Unit Price:
              </td>
              <td
                align="right"
                style="
                  padding-bottom: 10px;
                  font-size: 14px;
                  font-weight: 400;
                  color: #000;
                  padding: 10px 0;
                  border-bottom: 1px solid #ebebeb;
                "
              >
                $ %.2f
              </td>
            </tr>
			<tr>
              <td
                style="
                  padding-bottom: 10px;
                  font-size: 14px;
                  font-weight: 400;
                  color: #545454;
                  padding: 10px 0;
                  border-bottom: 1px solid #ebebeb;
                  white-space: nowrap;
                "
              >
                Sub Total:
              </td>
              <td
                align="right"
                style="
                  padding-bottom: 10px;
                  font-size: 14px;
                  font-weight: 400;
                  color: #000;
                  padding: 10px 0;
                  border-bottom: 1px solid #ebebeb;
                "
              >
                $ %.2f
              </td>
            </tr>
            <tr>
              <td
                style="
                  padding-bottom: 10px;
                  font-size: 14px;
                  font-weight: 400;
                  color: #545454;
                  padding: 10px 0;
                  border-bottom: 1px solid #ebebeb;
                  white-space: nowrap;
                "
              >
               GST (%.2f%%):
              </td>
              <td
                align="right"
                style="
                  padding-bottom: 10px;
                  font-size: 14px;
                  font-weight: 400;
                  color: #000;
                  padding: 10px 0;
                  border-bottom: 1px solid #ebebeb;
                "
              >
               $ %.2f
              </td>
            </tr>
            <tr>
              <td
                style="
                  padding-bottom: 10px;
                  font-size: 14px;
                  font-weight: 400;
                  color: #545454;
                  padding: 10px 0;
                  border-bottom: 1px solid #ebebeb;
                  white-space: nowrap;
                "
              >
                Discount (%.2f%%):
              </td>
              <td
                align="right"
                style="
                  padding-bottom: 10px;
                  font-size: 14px;
                  font-weight: 400;
                  color: #000;
                  padding: 10px 0;
                  border-bottom: 1px solid #ebebeb;
                "
              >
              $ %.2f
              </td>
            </tr>
            <tr>
              <td
                style="
                  padding-bottom: 10px;
                  font-size: 14px;
                  font-weight: 400;
                  color: #545454;
                  padding: 10px 0;
                  white-space: nowrap;
                "
              >
                Total Cost:
              </td>
              <td
                align="right"
                style="
                  padding-bottom: 10px;
                  font-size: 14px;
                  font-weight: 400;
                  color: #000;
                  padding: 10px 0;
                "
              >
               $ %.2f
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
                  href="%s/%s"
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
                  >Download Invoice</a
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
                
	`, orderNumber, utils.CapitalizeWords(fmt.Sprintf("%s %s", customerFirstName, customerLastName)), clinicIDSection, customerEmail, customerPhoneNumber, customerStreetAddress, customerTownCity, customerRegion, customerPostcode, customerCountry, customerShippingStreetAddress, customerShippingTownCity, customerShippingRegion, customerShippingPostcode, customerShippingCountry, orderNumber, productName, productDescription, productQuantity, productPrice, subTotal, gstPercentage, gstTotal, productDiscount, discountTotal, orderTotalAmount, apiUrl, invoiceLink)

	return CommonEmailTemplate(bodyContent)
}
