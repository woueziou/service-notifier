package handler

import (
	"fmt"
	"strings"

	"github.com/go-playground/validator/v10"
)

var validate *validator.Validate

func init() {
	validate = validator.New()
}

// ValidateStruct validates a struct using go-playground/validator tags.
// Returns a user-friendly error message or empty string if valid.
func ValidateStruct(s interface{}) string {
	if err := validate.Struct(s); err != nil {
		if errs, ok := err.(validator.ValidationErrors); ok {
			var msgs []string
			for _, e := range errs {
				msgs = append(msgs, fieldError(e))
			}
			return strings.Join(msgs, "; ")
		}
		return fmt.Sprintf("validation error: %s", err.Error())
	}
	return ""
}

func fieldError(e validator.FieldError) string {
	switch e.Tag() {
	case "required":
		return fmt.Sprintf("%s is required", e.Field())
	case "email":
		return fmt.Sprintf("%s must be a valid email address", e.Field())
	case "min":
		return fmt.Sprintf("%s must be at least %s characters", e.Field(), e.Param())
	case "max":
		return fmt.Sprintf("%s must be at most %s characters", e.Field(), e.Param())
	default:
		return fmt.Sprintf("%s failed validation: %s=%s", e.Field(), e.Tag(), e.Param())
	}
}
