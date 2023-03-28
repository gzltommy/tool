package middleware

import (
	"fmt"
	"github.com/dgrijalva/jwt-go"
	"testing"
	"time"
	"tool-attendance/types"
)

func TestAuthorized(t *testing.T) {
	code := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"id":      1,
		"address": "0x0f04f0d2de3c4b61a84658dbf973ac095865e3e3",
		"nbf":     time.Now().Unix(),
		"iat":     time.Now().Unix(),
		"exp":     time.Now().Add(time.Duration(90000000) * time.Second).Unix(),
	})
	token, _ := code.SignedString([]byte("Lbhi08lqB8k7bdGzfosSyZwPygIOvwhX"))
	fmt.Println(token)
}

func TestReadToken(t *testing.T) {
	accessToken := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJhZGRyZXNzIjoiMHgwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwIiwiZXhwIjoxNjY1ODE3MjYxLCJpYXQiOjE2NjU3MzA4NjEsImlkIjozMDEwOCwibmJmIjoxNjY1NzMwODYxfQ.g4r1iXDSQGf8toEzuzrpTTgnILJ9Qqh6ksGFu4lIZ9c"
	token, err := jwt.ParseWithClaims(accessToken, &types.AccountAuthClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			t.Error("token parse err")
		}
		return []byte("Lbhi08lqB8k7bdGzfosSyZwPygIOvwhX"), nil
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
		t.Log(errMsg)
		t.Error(err)
	}
	if claims, ok := token.Claims.(*types.AccountAuthClaims); ok && token.Valid {
		t.Log(claims.ID)
		t.Log(claims.Id)
	}
}
