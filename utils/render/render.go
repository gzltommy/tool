package render

import (
	"fmt"
	"net/http"
	"tool-attendance/log"

	"github.com/gin-gonic/gin"
)

type RespJsonData struct {
	Code int         `json:"code"`
	Msg  string      `json:"message"`
	Data interface{} `json:"data"`
}

func AbortJson(c *gin.Context, code int, data interface{}) {
	msg := getMessage(code)
	result := &RespJsonData{
		Code: code,
		Msg:  msg,
		Data: data,
	}
	c.AbortWithStatusJSON(code, result)
}

func Json(c *gin.Context, code int, data interface{}) {
	if 100 <= code && code <= 600 {
		if code == 200 {
			code = 0
		} else {
			code = UnKnowError
		}
	}
	msg := getMessage(code)
	if code != Ok {
		if log.Log != nil {
			log.Log.Errorf("Request:%s, Code:%d, Msg:%s,Data:%v", c.Request.RequestURI, code, msg, data)
		} else {
			fmt.Printf("Request:%s, Code:%d, Msg:%s,Data:%v\n", c.Request.RequestURI, code, msg, data)
		}
	}
	result := &RespJsonData{
		Code: code,
		Msg:  msg,
		Data: data,
	}
	c.JSON(http.StatusOK, result)
}

func getMessage(code int) string {
	msg := ""
	if 100 <= code && code <= 600 {
		msg = http.StatusText(code)
	} else {
		msg = statusMsg[code]
	}
	return msg
}
