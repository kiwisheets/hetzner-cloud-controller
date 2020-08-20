package auth

import (
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/emvi/hide"
	"github.com/kiwisheets/hetzner-cloud-controller/model"
)

// UserClaim structure
type UserClaim struct {
	UserID hide.ID `json:"userId"`
	jwt.StandardClaims
}

func ValidateTokenAndGetUserID(t string, jwtSecret string) (hide.ID, error) {
	token, err := jwt.ParseWithClaims(t, &UserClaim{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(jwtSecret), nil
	})

	if err != nil {
		return 0, err
	}

	if claims, ok := token.Claims.(*UserClaim); ok && token.Valid && token.Method == jwt.SigningMethodHS384 {
		return claims.UserID, nil
	}
	return 0, err
}

// buildAndSignToken signs and returned a JWT token from a User
func buildAndSignToken(u *model.User, jwtSecret string, expires time.Duration) (string, error) {
	claims := UserClaim{
		UserID: u.ID,
		StandardClaims: jwt.StandardClaims{
			Issuer:   "HCC",
			IssuedAt: time.Now().Unix(),
			Subject:  "Hetzner Cloud Controller",
		},
	}

	if expires != 0 {
		claims.ExpiresAt = time.Now().Add(expires).Unix()
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS384, claims)
	return token.SignedString([]byte(jwtSecret))
}
