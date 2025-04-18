package utils

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/go-playground/validator/v10"
)

// Validate validates a struct
func Validate(s interface{}) (map[string]string, error) {
	validate := validator.New()
	validate.RegisterTagNameFunc(func(fld reflect.StructField) string {
		name := strings.SplitN(fld.Tag.Get("json"), ",", 2)[0]
		if name == "-" {
			return ""
		}
		return name
	})

	err := validate.Struct(s)
	if err == nil {
		return nil, nil
	}

	errors := make(map[string]string)
	for _, err := range err.(validator.ValidationErrors) {
		field := err.Field()
		errors[field] = getErrorMsg(err)
	}

	return errors, err
}

// getErrorMsg returns a human-readable error message
func getErrorMsg(e validator.FieldError) string {
	switch e.Tag() {
	case "required":
		return "This field is required"
	case "email":
		return "Please enter a valid email address"
	case "min":
		return fmt.Sprintf("This field must be at least %s characters long", e.Param())
	case "max":
		return fmt.Sprintf("This field must not exceed %s characters", e.Param())
	case "alphanum":
		return "This field must contain only letters and numbers"
	case "oneof":
		return fmt.Sprintf("This field must be one of: %s", e.Param())
	default:
		return fmt.Sprintf("This field failed validation: %s", e.Tag())
	}
}
