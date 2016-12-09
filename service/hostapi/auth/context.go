package auth

import (
	jwt "github.com/dgrijalva/jwt-go"
	"github.com/gorilla/context"
	"net/http"
)

type key int

const TokenKey key = 0

func GetToken(r *http.Request) *jwt.Token {
	if rv := context.Get(r, TokenKey); rv != nil {
		return rv.(*jwt.Token)
	}
	return nil
}

func SetToken(r *http.Request, val *jwt.Token) {
	context.Set(r, TokenKey, val)
}
