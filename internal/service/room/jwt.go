package room

import (
	"errors"

	"github.com/golang-jwt/jwt/v5"
)

type Claims struct {
	MemberId string `json:"member_id"`
}

func (s service) generateJWT(memberId string) (string, error) {
	claims := jwt.MapClaims{
		"member_id": memberId,
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
		MemberId: claims["member_id"].(string),
	}, nil
}
