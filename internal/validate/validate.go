package validate

import (
	"fmt"

	"github.com/go-playground/validator/v10"
)

var v = validator.New(validator.WithRequiredStructEnabled())

// Map returns field->message errors for struct validation tags.
func Map(s any) map[string]string {
	if err := v.Struct(s); err != nil {
		if verrs, ok := err.(validator.ValidationErrors); ok {
			m := make(map[string]string, len(verrs))
			for _, fe := range verrs {
				m[fieldName(fe)] = messageFor(fe)
			}
			return m
		}
		return map[string]string{"_error": err.Error()}
	}
	return nil
}

func fieldName(fe validator.FieldError) string {
	// Use json field if available; fallback to struct field name
	if fe.Field() != "" {
		return toLowerFirst(fe.Field())
	}
	return fe.StructField()
}

func toLowerFirst(s string) string {
	if s == "" {
		return s
	}
	r := []rune(s)
	r[0] = []rune(string(r[0]))[0] + 32
	return string(r)
}

func messageFor(fe validator.FieldError) string {
	switch fe.Tag() {
	case "required":
		return "is required"
	case "min":
		return fmt.Sprintf("must be at least %s characters", fe.Param())
	case "max":
		return fmt.Sprintf("must be at most %s characters", fe.Param())
	case "gte":
		return fmt.Sprintf("must be >= %s", fe.Param())
	case "lte":
		return fmt.Sprintf("must be <= %s", fe.Param())
	case "oneof":
		return fmt.Sprintf("must be one of %s", fe.Param())
	case "gt":
		return fmt.Sprintf("must be > %s", fe.Param())
	default:
		return fe.Error()
	}
}
