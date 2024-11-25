package validator

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/go-playground/validator/v10"
)

type ValidationError struct {
	Field   string `json:"field" yaml:"field" xml:"field" bson:"field"`
	Code    string `json:"code" yaml:"code" xml:"code" bson:"code"`
	Message string `json:"message" yaml:"message" xml:"message" bson:"message"`
}

type Validator struct {
	validate *validator.Validate
}

func NewValidator() *Validator {
	v := validator.New()
	v.RegisterTagNameFunc(func(fld reflect.StructField) string {
		name := strings.SplitN(fld.Tag.Get("json"), ",", 2)[0]

		if name == "-" {
			return ""
		}

		return name
	})

	return &Validator{validate: v}
}

func (v *Validator) Validate(i any) ([]ValidationError, bool) {
	if err := v.validate.Struct(i); err != nil {
		validationErrors := err.(validator.ValidationErrors)
		errors := make([]ValidationError, 0, len(validationErrors))

		for _, err := range validationErrors {
			var message string
			switch err.Tag() {
			case "required":
				message = fmt.Sprintf("%s is required", err.Field())
			case "min":
				message = fmt.Sprintf("%s must be at least %s characters long", err.Field(), err.Param())
			case "max":
				message = fmt.Sprintf("%s must not exceed %s characters", err.Field(), err.Param())
			}

			errors = append(errors, ValidationError{
				Field: err.Field(),
				// todo: change codes and wrap in enums
				Code:    strings.ToUpper(err.Tag()),
				Message: message,
			})
		}

		return errors, false
	}

	return nil, true
}
