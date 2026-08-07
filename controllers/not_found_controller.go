// controllers/not_found_controller.go
package controllers

import (
	"net/http"
	"theransticslabs/m/utils"

	"github.com/gin-gonic/gin"
)

func NotFoundHandler(c *gin.Context) {
	utils.JSONResponse(c, http.StatusNotFound, utils.MsgEndpointNotFound, nil)
}
