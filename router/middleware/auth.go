package middleware

import (
	"errors"
	"net/http"
	"strings"
	"tool-attendance/config"

	"tool-attendance/types"
	"tool-attendance/utils/render"

	jwt "github.com/dgrijalva/jwt-go"
	"github.com/gin-gonic/gin"
)

// 管理后台的token验证
func Authorized(c *gin.Context) {
	var accessToken string
	secret := config.GetConfig().App.Secret
	authorizationHeader := c.Request.Header.Get("Authorization")
	getToken := c.DefaultQuery("token", "")
	if authorizationHeader == "" && getToken == "" {
		render.AbortJson(c, http.StatusUnauthorized, "Authorization header not provided")
		return
	}
	header := strings.Split(authorizationHeader, " ")
	if len(header) == 2 && header[0] == "Bearer" {
		accessToken = header[1]
	} else {
		accessToken = getToken
	}
	token, err := jwt.ParseWithClaims(accessToken, &types.AuthClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("Token parse error")
		}
		return []byte(secret), nil
	})
	if err != nil {
		var errMsg string
		if ve, ok := err.(*jwt.ValidationError); ok {
			if ve.Errors&jwt.ValidationErrorMalformed != 0 {
				errMsg = "The token not active yet"
			} else if ve.Errors&jwt.ValidationErrorExpired != 0 {
				errMsg = "The token is expired"
			} else if ve.Errors&jwt.ValidationErrorNotValidYet != 0 {
				errMsg = "The token is error"
			} else {
				errMsg = "The token unknown error"
			}
		}
		render.AbortJson(c, http.StatusUnauthorized, errMsg)
		return
	}
	if claims, ok := token.Claims.(*types.AuthClaims); ok && token.Valid {
		c.Set("claims", claims)
		c.Next()
	} else {
		render.AbortJson(c, http.StatusUnauthorized, "Authorization Bearer is error")
		return
	}
}

// 前台用户的token验证
func AccountAuthorized(c *gin.Context) {
	var accessToken string
	secret := config.GetConfig().App.Secret
	authorizationHeader := c.Request.Header.Get("Authorization")
	platform := c.Request.Header.Get("platform")
	if platform == "" {
		platform = "web"
	}
	getToken := c.DefaultQuery("token", "")
	if authorizationHeader == "" && getToken == "" {
		abortJson(c, platform, "Authorization header not provided")
		return
	}
	header := strings.Split(authorizationHeader, " ")
	if len(header) == 2 && header[0] == "Bearer" {
		accessToken = header[1]
	} else {
		accessToken = getToken
	}

	token, err := jwt.ParseWithClaims(accessToken, &types.AccountAuthClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("Token parse error")
		}
		return []byte(secret), nil
	})
	if err != nil {
		var errMsg string
		if ve, ok := err.(*jwt.ValidationError); ok {
			if ve.Errors&jwt.ValidationErrorMalformed != 0 {
				errMsg = "The token not active yet"
			} else if ve.Errors&jwt.ValidationErrorExpired != 0 {
				errMsg = "The token is expired"
			} else if ve.Errors&jwt.ValidationErrorNotValidYet != 0 {
				errMsg = "The token is error"
			} else {
				errMsg = "The token unknown error"
			}
		}
		abortJson(c, platform, errMsg)
		return
	}
	if claims, ok := token.Claims.(*types.AccountAuthClaims); ok && token.Valid {
		c.Set("claims", claims)
		c.Next()
	} else {
		abortJson(c, platform, "Authorization Bearer is error")
		return
	}
}

func abortJson(c *gin.Context, platform string, msg string) {
	if platform == "web" {
		render.AbortJson(c, http.StatusUnauthorized, msg)
	} else {
		c.JSON(http.StatusOK, render.RespJsonData{
			Code: render.ErrForbidden,
			Msg:  "The login is invalid, please relogin.",
			Data: nil,
		})
		c.Abort()
	}
}
