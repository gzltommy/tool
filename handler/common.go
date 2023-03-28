package handler

import (
	"github.com/gin-gonic/gin"
	"tool-attendance/utils/render"
)

func Pong(c *gin.Context) {
	render.Json(c, render.Ok, "pong!")
}
