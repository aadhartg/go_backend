package controllers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"theransticslabs/m/utils"

	"github.com/gin-gonic/gin"
)

type Product struct {
	Name        string  `json:"name" validate:"required" form:"name"`
	Description string  `json:"description" form:"description"`
	Image       string  `json:"image" form:"image"`
	Price       float64 `json:"price" validate:"required,gt=0" form:"price"`
	GstPrice    float64 `json:"gst_price" validate:"required,gt=0" form:"gst_price"`
}

type EncryptedData struct {
	Data string `json:"data" validate:"required" form:"data"`
}

// EncryptProductDetails handles the encryption of product details
func EncryptProductDetails(c *gin.Context) {
	// Parse the request body
	var req Product

	// Use the common request parser
	// Define allowed fields for this request
	allowedFields := []string{"name", "description", "image", "price", "gst_price"}

	err := utils.ParseRequestBody(c.Request, &req, allowedFields) // Assuming this validates required fields
	if err != nil {
		utils.JSONResponse(c, http.StatusBadRequest, err.Error(), nil)
		return
	}

	// Marshal struct to JSON
	jsonData, err := json.Marshal(req)
	if err != nil {
		utils.JSONResponse(c, http.StatusInternalServerError, utils.MsgErrorEncodingProductDetails, nil)
		return
	}

	// Encrypt the JSON string
	encryptedData, err := utils.Encrypt(string(jsonData))
	if err != nil {
		utils.JSONResponse(c, http.StatusInternalServerError, utils.MsgErrorEncryptingProductDetails, map[string]string{"error": err.Error()})
		return
	}
	// Success response
	utils.JSONResponse(c, http.StatusOK, utils.MsgProductDetailsEncryptedSuccessfully, encryptedData)
}

// VerifyProduct validates and verifies an encrypted product data request
func VerifyProduct(c *gin.Context) {
	// Parse the request body into EncryptedData struct
	var req EncryptedData

	// Define allowed fields for this request
	allowedFields := []string{"data"}

	// Use the common request parser to validate allowed fields
	if err := utils.ParseRequestBody(c.Request, &req, allowedFields); err != nil {
		utils.JSONResponse(c, http.StatusBadRequest, err.Error(), nil)
		return
	}

	// Validate that 'data' is present and non-empty in the request
	if req.Data == "" {
		utils.JSONResponse(c, http.StatusBadRequest, utils.MsgMissingOrEmptyEncryptedDataParameter, nil)
		return
	}

	// Attempt to decrypt the encrypted data
	decryptedData, err := utils.Decrypt(req.Data)
	if err != nil {
		utils.JSONResponse(c, http.StatusBadRequest, utils.MsgFailedToDecryptData, map[string]string{"error": err.Error()})
		return
	}

	// Unmarshal decrypted JSON into a Product struct
	var product Product
	if err := json.Unmarshal([]byte(decryptedData), &product); err != nil {
		utils.JSONResponse(c, http.StatusBadRequest, utils.MsgInvalidProductDataFormat, map[string]string{"error": err.Error()})
		return
	}

	// Validate required fields in the decrypted product data
	// Validate product name
	if product.Name == "" || !utils.IsValidProductName(product.Name) {
		utils.JSONResponse(c, http.StatusBadRequest, utils.MsgInvalidProductName, nil)
		return
	}

	// Validate product price
	if product.Price <= 0 || !utils.IsValidPrice(strconv.FormatFloat(product.Price, 'f', 2, 64)) {
		utils.JSONResponse(c, http.StatusBadRequest, utils.MsgProductPriceMustBeGreaterThanZero, product)
		return
	}

	// Validate product price
	if product.GstPrice <= 0 || !utils.IsValidPrice(strconv.FormatFloat(product.GstPrice, 'f', 2, 64)) {
		utils.JSONResponse(c, http.StatusBadRequest, utils.MsgProductGstPriceMustBeGreaterThanZero, product)
		return
	}

	// Validate product description if provided
	if product.Description != "" && len(product.Description) > 1000 {
		utils.JSONResponse(c, http.StatusBadRequest, utils.MsgProductDescriptionTooLong, nil)
		return
	}

	// Validate product image as either base64 or URL
	if product.Image != "" && !(utils.IsValidBase64Image(product.Image) || utils.IsValidImageURL(product.Image)) {
		utils.JSONResponse(c, http.StatusBadRequest, utils.MsgInvalidImageFormat, nil)
		return
	}

	// Return the decrypted and validated product data
	utils.JSONResponse(c, http.StatusOK, utils.MsgProductVerifiedSuccessfully, product)
}
