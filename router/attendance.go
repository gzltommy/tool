package router

import (
	"github.com/gin-gonic/gin"
	"tool-attendance/config"
	"tool-attendance/handler"
)

func InitAiRouter(cfg *config.Configuration) *gin.Engine {
	r := initDefaultRouter(cfg)
	v1 := r.Group("/api/v1")
	v1.GET("/ping", handler.Pong)
	{
		v1.GET("/init/calendar/:year", handler.InitCalendar)
		v1.GET("/attendance/detail/:month", handler.AttendanceDetail)
		v1.GET("/attendance/record/:month", handler.AttendanceRecord)
	}
	return r
}
