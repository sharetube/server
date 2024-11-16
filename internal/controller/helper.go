package controller

import (
	"fmt"
	"net/http"
)

const (
	headerPrefix = "St-"
)

func (c Controller) MustHeader(r *http.Request, key string) (string, error) {
	value := r.Header.Get(headerPrefix + key)
	if value == "" {
		return "", fmt.Errorf("%s was not provided", key)
	}

	return value, nil
}
