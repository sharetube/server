package service

import (
	"errors"

	"github.com/golang-jwt/jwt/v5"
)

var ErrInvalidToken = errors.New("invalid token")

const memberIdKey = "member_id"

type Claims struct {
	MemberId string `json:"member_id"`
}

func (s service) generateJWT(memberId string) (string, error) {
	claims := jwt.MapClaims{
		memberIdKey: memberId,
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
		return nil, ErrInvalidToken
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, ErrInvalidToken
	}

	memberId, ok := claims[memberIdKey].(string)
	if !ok {
		return nil, ErrInvalidToken
	}

	return &Claims{
		MemberId: memberId,
	}, nil
}
