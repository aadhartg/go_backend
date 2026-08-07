// utils/paypal.go

package utils

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"strconv"
	"strings"
	"theransticslabs/m/config"
	"theransticslabs/m/models"
)

// Define the response structure for PayPal access token
type PayPalAccessTokenResponse struct {
	Scope       string `json:"scope"`
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	AppID       string `json:"app_id"`
	ExpiresIn   int    `json:"expires_in"`
	Nonce       string `json:"nonce"`
}

// Define the response structure for PayPal order
type PayPalOrderResponse struct {
	ID     string `json:"id"`
	Status string `json:"status"`
	Links  []struct {
		Href   string `json:"href"`
		Rel    string `json:"rel"`
		Method string `json:"method"`
	} `json:"links"`
}

// PayPalCaptureResponse represents the response from PayPal's capture endpoint
type PayPalCaptureResponse struct {
	ID            string `json:"id"`
	Status        string `json:"status"`
	PaymentSource struct {
		PayPal struct {
			EmailAddress string `json:"email_address"`
			AccountID    string `json:"account_id"`
		} `json:"paypal"`
	} `json:"payment_source"`
}

func GetPayPalAccessToken() (string, error) {
	url := config.AppConfig.PaypalAPIUrl + "/v1/oauth2/token"
	clientID := config.AppConfig.PaypalClientID
	secret := config.AppConfig.PaypalClientSecret

	req, err := http.NewRequest("POST", url, bytes.NewBuffer([]byte("grant_type=client_credentials")))
	if err != nil {
		return "", err
	}

	req.SetBasicAuth(clientID, secret)
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var res PayPalAccessTokenResponse
	json.NewDecoder(resp.Body).Decode(&res)

	if res.AccessToken == "" {
		return "", errors.New("failed to get PayPal token")
	}
	return res.AccessToken, nil
}

// Helper function to sanitize and truncate strings
func sanitizeString(s string, maxLength int) string {
	// Remove any non-printable characters
	s = strings.Map(func(r rune) rune {
		if r < 32 || r > 126 {
			return -1
		}
		return r
	}, s)

	// Truncate to maxLength if necessary
	if len(s) > maxLength {
		s = s[:maxLength]
	}

	return s
}

// Helper function to verify amount breakdown
func verifyAmountBreakdown(subtotal, gst, discount, total float64) error {
	// Precise calculation with rounding
	calculatedTotal := math.Round((subtotal+gst-discount)*100) / 100

	if math.Abs(calculatedTotal-total) > 0.01 {
		return fmt.Errorf("amount mismatch: calculated total %.2f does not match provided total %.2f",
			calculatedTotal, total)
	}

	return nil
}

func CreatePayPalOrder(paymentID uint, amount float64, accessToken string, orderDetails *models.Order, customer *models.Customer) (PayPalOrderResponse, error) {
	url := config.AppConfig.PaypalAPIUrl + "/v2/checkout/orders"

	returnURL := fmt.Sprintf("%s/payment-status?payment_id=%d&action=success", config.AppConfig.AppUrl, paymentID)
	cancelURL := fmt.Sprintf("%s/payment-status?payment_id=%d&action=cancel", config.AppConfig.AppUrl, paymentID)

	// Precise calculation with rounding to avoid floating-point inaccuracies
	unitPrice := orderDetails.ProductPrice
	quantity := float64(orderDetails.Quantity)

	// Calculate subtotal exactly
	subtotal := math.Round(unitPrice*quantity*100) / 100

	// Calculate discount exactly
	discountRate := orderDetails.ProductDiscount / 100
	discountAmount := math.Round(subtotal*discountRate*100) / 100

	// Calculate GST exactly
	gstPrice := orderDetails.ProductGstPrice
	totalGst := math.Round(gstPrice*quantity*100) / 100

	// Calculate total amount exactly
	totalAmount := math.Round((subtotal+totalGst-discountAmount)*100) / 100

	payload := map[string]interface{}{
		"intent": "CAPTURE",
		"application_context": map[string]interface{}{
			"return_url":          returnURL,
			"cancel_url":          cancelURL,
			"shipping_preference": "NO_SHIPPING",
			"user_action":         "PAY_NOW",
			"brand_name":          sanitizeString("Theranostics DNA", 127),

			"payment_method": map[string]interface{}{
				"payer_selected":            "PAYPAL",
				"payee_preferred":           "IMMEDIATE_PAYMENT_REQUIRED",
				"standard_entry_class_code": "WEB",
			},
		},

		"purchase_units": []map[string]interface{}{
			{
				"reference_id": strconv.FormatUint(uint64(paymentID), 10),
				"description":  fmt.Sprintf("Order #%d", orderDetails.ID),
				"custom_id":    fmt.Sprintf("ORDER_%d", orderDetails.ID),
				"amount": map[string]interface{}{
					"currency_code": "USD",
					"value":         fmt.Sprintf("%.2f", totalAmount),
					"breakdown": map[string]interface{}{
						"item_total": map[string]string{
							"currency_code": "USD",
							"value":         fmt.Sprintf("%.2f", subtotal),
						},
						"tax_total": map[string]string{
							"currency_code": "USD",
							"value":         fmt.Sprintf("%.2f", totalGst),
						},
						"shipping": map[string]string{
							"currency_code": "USD",
							"value":         "0.00",
						},
						"discount": map[string]string{
							"currency_code": "USD",
							"value":         fmt.Sprintf("%.2f", discountAmount),
						},
					},
				},
				"items": []map[string]interface{}{
					{
						"name":        orderDetails.ProductName,
						"description": fmt.Sprintf("Product Details: %s", orderDetails.ProductName),
						"quantity":    strconv.Itoa(orderDetails.Quantity),
						"unit_amount": map[string]string{
							"currency_code": "USD",
							"value":         fmt.Sprintf("%.2f", unitPrice),
						},
					},
				},
			},
		},
	}

	// Verify amount calculations before sending
	verifyAmountBreakdown(subtotal, totalGst, discountAmount, totalAmount)

	payloadBytes, _ := json.Marshal(payload)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(payloadBytes))
	if err != nil {
		return PayPalOrderResponse{}, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+accessToken)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return PayPalOrderResponse{}, err
	}
	defer resp.Body.Close()

	var res PayPalOrderResponse
	json.NewDecoder(resp.Body).Decode(&res)
	return res, nil
}

// CapturePayPalPayment captures a previously authorized PayPal payment
func CapturePayPalPayment(orderID string, accessToken string) error {
	url := fmt.Sprintf("%s/v2/checkout/orders/%s/capture", config.AppConfig.PaypalAPIUrl, orderID)

	// Create request
	req, err := http.NewRequest("POST", url, bytes.NewBuffer([]byte{}))
	if err != nil {
		return fmt.Errorf("failed to create capture request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", accessToken))
	req.Header.Set("Prefer", "return=representation")

	// Send request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send capture request: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read capture response: %w", err)
	}

	// Check for successful status code
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("failed to capture payment. Status: %d, Body: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var captureResponse PayPalCaptureResponse
	if err := json.Unmarshal(body, &captureResponse); err != nil {
		return fmt.Errorf("failed to parse capture response: %w", err)
	}

	// Verify capture status
	if captureResponse.Status != "COMPLETED" {
		return fmt.Errorf("payment capture failed. Status: %s", captureResponse.Status)
	}

	return nil
}
