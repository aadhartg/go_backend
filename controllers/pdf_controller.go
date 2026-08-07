package controllers

import (
	"net/http"
	"theransticslabs/m/dto"
	"theransticslabs/m/utils"

	"github.com/gin-gonic/gin"
)

// GenerateConsentPDFHandler handles POST /generate-pdf and returns a PDF file
func GenerateConsentPDFHandler(c *gin.Context) {
	var req dto.ConsentFormPDFRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.JSONResponse(c, http.StatusBadRequest, "Invalid input: "+err.Error(), nil)
		return
	}

	pdfBytes, err := utils.GenerateConsentPDF(req)
	if err != nil {
		utils.JSONResponse(c, http.StatusInternalServerError, "Failed to generate PDF: "+err.Error(), nil)
		return
	}

	c.Header("Content-Type", "application/pdf")
	c.Header("Content-Disposition", "attachment; filename=consent_form.pdf")
	c.Data(http.StatusOK, "application/pdf", pdfBytes)
} 