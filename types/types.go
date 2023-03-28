package types

import (
	"github.com/dgrijalva/jwt-go"
)

type AuthClaims struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
	jwt.StandardClaims
}

type UserClaims struct {
	ID int64 `json:"id"`
}

type AccountAuthClaims struct {
	ID      int64  `json:"id"`
	Address string `json:"address"`
	jwt.StandardClaims
}
