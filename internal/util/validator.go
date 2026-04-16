package util

import (
	"errors"
	"fmt"
	"strings"

	"github.com/go-playground/validator/v10"
)

// ParseValidationErrors translates validator.ValidationErrors into a user-friendly map.
func ParseValidationErrors(err error) map[string]string {
	var ve validator.ValidationErrors
	if errors.As(err, &ve) {
		out := make(map[string]string)
		for _, fe := range ve {
			out[strings.ToLower(fe.Field())] = getErrorMsg(fe)
		}
		return out
	}
	return nil
}

// getErrorMsg translates a single FieldError tag into a user-friendly message.
func getErrorMsg(fe validator.FieldError) string {
	switch fe.Tag() {
	case "required":
		return "is required and cannot be empty"
	case "email":
		return "must be a valid email address"
	case "len":
		return fmt.Sprintf("must be exactly %s characters long", fe.Param())
	case "min":
		return fmt.Sprintf("must be at least %s", fe.Param())
	case "max":
		return fmt.Sprintf("must be at most %s", fe.Param())
	case "uuid":
		return "must be a valid UUID"
	case "numeric":
		return "must be a valid number"
	case "alphanum":
		return "must contain only letters and numbers"
	}
	return fmt.Sprintf("failed validation on the %s tag", fe.Tag())
}
