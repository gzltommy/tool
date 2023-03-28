package router

import (
	"net/http"
	"time"

	limit "github.com/aviddiviner/gin-limit"
	"github.com/gin-contrib/cors"
	"github.com/gin-contrib/gzip"
	"github.com/gin-gonic/gin"

	"tool-attendance/config"
)

func initDefaultRouter(cfg *config.Configuration) *gin.Engine {
	r := gin.Default()
	r.Use(gin.Recovery())
	r.Use(gzip.Gzip(gzip.DefaultCompression))
	//corsConfig := getCorsConfig()
	//corsConfig.AllowAllOrigins = true
	//r.Use(cors.New(corsConfig))
	r.Use(Cors())
	if cfg.Server.LimitConnection > 0 {
		r.Use(limit.MaxAllowed(cfg.Server.LimitConnection))
	}
	r.HandleMethodNotAllowed = true
	r.NoMethod(func(c *gin.Context) {
		c.JSON(http.StatusMethodNotAllowed, gin.H{
			"result": false,
			"error":  "Method Not Allowed",
		})
		return
	})
	r.NoRoute(func(c *gin.Context) {
		c.JSON(http.StatusNotFound, gin.H{
			"result": false,
			"error":  "Endpoint Not Found",
		})
		return
	})
	// 最大运行上传文件大小
	r.MaxMultipartMemory = 1024 * 1024 * 1024 //1G
	return r
}

func getCorsConfig() cors.Config {
	return cors.Config{
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD"},
		AllowHeaders:     []string{"Origin", "Pragma", "Set-Cookie", "Cache-Control", "Connection", "Content-Length", "Content-Type", "Authorization", "X-Forwarded-For", "User-Agent", "Referer"},
		AllowCredentials: true,
		ExposeHeaders:    []string{"Set-Cookie"},
		MaxAge:           12 * time.Hour,
	}
}

func Cors() gin.HandlerFunc {
	return func(c *gin.Context) {
		method := c.Request.Method //请求方法

		c.Header("Access-Control-Allow-Origin", "*")                                                  // 这是允许访问所有域
		c.Header("Access-Control-Allow-Methods", "POST, GET, OPTIONS,HEAD, PUT, PATCH,DELETE,UPDATE") //服务器支持的所有跨域请求的方法,为了避免浏览次请求的多次'预检'请求
		//  header的类型
		c.Header("Access-Control-Allow-Headers", "Authorization, X-Forwarded-For,Content-Length, X-CSRF-Token, Token,session,X_Requested_With,Accept, Origin, Host, Connection, Accept-Encoding, Accept-Language,DNT, X-CustomHeader, Keep-Alive, User-Agent, X-Requested-With, If-Modified-Since, Cache-Control, Content-Type, Pragma, platform")

		//              允许跨域设置                                                                                                      可以返回其他子段
		c.Header("Access-Control-Expose-Headers", "Content-Length, Access-Control-Allow-Origin, Access-Control-Allow-Headers,Cache-Control,Content-Language,Content-Type,Expires,Last-Modified,Pragma,FooBar,Origin") // 跨域关键设置 让浏览器可以解析
		c.Header("Access-Control-Max-Age", "86400")                                                                                                                                                                   // 缓存请求信息 单位为秒
		c.Header("Access-Control-Allow-Credentials", "false")                                                                                                                                                         //  跨域请求是否需要带cookie信息 默认设置为true
		c.Set("content-type", "application/json")                                                                                                                                                                     // 设置返回格式是json

		//放行所有OPTIONS方法
		if method == "OPTIONS" {
			c.JSON(http.StatusOK, "Options Request!")
		}
		// 处理请求
		c.Next() //  处理请求
	}
}
