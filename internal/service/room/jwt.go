package room

import (
	"errors"

	"github.com/golang-jwt/jwt/v5"
)

type Claims struct {
	MemberID string `json:"member_id"`
}

func (s service) generateJWT(memberID string) (string, error) {
	claims := jwt.MapClaims{
		"member_id": memberID,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	return token.SignedString([]byte(s.secret))
}

func (s service) parseJWT(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, jwt.MapClaims{}, func(token *jwt.Token) (interface{}, error) {
		return s.secret, nil
	})
	if err != nil {
		return nil, err
	}

	if !token.Valid {
		return nil, errors.New("invalid token")
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, errors.New("invalid token")
	}
	return &Claims{
		MemberID: claims["member_id"].(string),
	}, nil
}
