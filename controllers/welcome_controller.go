// controllers/welcome_controller.go
package controllers

import (
	"net/http"
	"theransticslabs/m/utils"

	"github.com/gin-gonic/gin"
)

func WelcomeHandler(c *gin.Context) {
	utils.JSONResponse(c, http.StatusOK, utils.MsgWelcome, nil)
}
