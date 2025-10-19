package validation

import (
	"fmt"
	"net/mail"
	"net/url"
	"reflect"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/conduit-lang/conduit/internal/orm/schema"
)

// Pre-compiled regex patterns for validators
var (
	e164Pattern = regexp.MustCompile(`^\+[1-9]\d{1,14}$`)
)

// Validator defines the interface for field validators
type Validator interface {
	Validate(value interface{}) error
}

// MinValidator validates minimum values for numeric types and string lengths
type MinValidator struct {
	Min       interface{}
	FieldType schema.PrimitiveType
}

// Validate implements the Validator interface
func (v *MinValidator) Validate(value interface{}) error {
	if value == nil {
		return nil // Nullable fields are validated separately
	}

	switch v.FieldType {
	case schema.TypeInt, schema.TypeBigInt:
		intVal, ok := toInt64(value)
		if !ok {
			return fmt.Errorf("expected integer value")
		}
		minVal, ok := toInt64(v.Min)
		if !ok {
			return fmt.Errorf("invalid min constraint")
		}
		if intVal < minVal {
			return fmt.Errorf("must be at least %d", minVal)
		}

	case schema.TypeFloat, schema.TypeDecimal:
		floatVal, ok := toFloat64(value)
		if !ok {
			return fmt.Errorf("expected numeric value")
		}
		minVal, ok := toFloat64(v.Min)
		if !ok {
			return fmt.Errorf("invalid min constraint")
		}
		if floatVal < minVal {
			return fmt.Errorf("must be at least %v", minVal)
		}

	case schema.TypeString, schema.TypeText, schema.TypeMarkdown, schema.TypeEmail, schema.TypeURL:
		strVal, ok := value.(string)
		if !ok {
			return fmt.Errorf("expected string value")
		}
		minLen, ok := toInt64(v.Min)
		if !ok {
			return fmt.Errorf("invalid min constraint")
		}
		if int64(utf8.RuneCountInString(strVal)) < minLen {
			return fmt.Errorf("must be at least %d characters", minLen)
		}
	}

	return nil
}

// MaxValidator validates maximum values for numeric types and string lengths
type MaxValidator struct {
	Max       interface{}
	FieldType schema.PrimitiveType
}

// Validate implements the Validator interface
func (v *MaxValidator) Validate(value interface{}) error {
	if value == nil {
		return nil // Nullable fields are validated separately
	}

	switch v.FieldType {
	case schema.TypeInt, schema.TypeBigInt:
		intVal, ok := toInt64(value)
		if !ok {
			return fmt.Errorf("expected integer value")
		}
		maxVal, ok := toInt64(v.Max)
		if !ok {
			return fmt.Errorf("invalid max constraint")
		}
		if intVal > maxVal {
			return fmt.Errorf("must be at most %d", maxVal)
		}

	case schema.TypeFloat, schema.TypeDecimal:
		floatVal, ok := toFloat64(value)
		if !ok {
			return fmt.Errorf("expected numeric value")
		}
		maxVal, ok := toFloat64(v.Max)
		if !ok {
			return fmt.Errorf("invalid max constraint")
		}
		if floatVal > maxVal {
			return fmt.Errorf("must be at most %v", maxVal)
		}

	case schema.TypeString, schema.TypeText, schema.TypeMarkdown, schema.TypeEmail, schema.TypeURL:
		strVal, ok := value.(string)
		if !ok {
			return fmt.Errorf("expected string value")
		}
		maxLen, ok := toInt64(v.Max)
		if !ok {
			return fmt.Errorf("invalid max constraint")
		}
		if int64(utf8.RuneCountInString(strVal)) > maxLen {
			return fmt.Errorf("must be at most %d characters", maxLen)
		}
	}

	return nil
}

// PatternValidator validates string values against a regex pattern
type PatternValidator struct {
	Pattern *regexp.Regexp
}

// Validate implements the Validator interface
func (v *PatternValidator) Validate(value interface{}) error {
	if value == nil {
		return nil
	}

	strVal, ok := value.(string)
	if !ok {
		return fmt.Errorf("pattern validation requires string value")
	}

	if !v.Pattern.MatchString(strVal) {
		return fmt.Errorf("does not match required pattern")
	}

	return nil
}

// EmailValidator validates email addresses
type EmailValidator struct{}

// Validate implements the Validator interface
func (v *EmailValidator) Validate(value interface{}) error {
	if value == nil {
		return nil
	}

	strVal, ok := value.(string)
	if !ok {
		return fmt.Errorf("email validation requires string value")
	}

	if strings.TrimSpace(strVal) == "" {
		return fmt.Errorf("email address cannot be empty")
	}

	// Use net/mail for RFC 5322 compliant email validation
	_, err := mail.ParseAddress(strVal)
	if err != nil {
		return fmt.Errorf("must be a valid email address")
	}

	return nil
}

// URLValidator validates URLs
type URLValidator struct{}

// Validate implements the Validator interface
func (v *URLValidator) Validate(value interface{}) error {
	if value == nil {
		return nil
	}

	strVal, ok := value.(string)
	if !ok {
		return fmt.Errorf("URL validation requires string value")
	}

	if strings.TrimSpace(strVal) == "" {
		return fmt.Errorf("URL cannot be empty")
	}

	parsedURL, err := url.Parse(strVal)
	if err != nil {
		return fmt.Errorf("must be a valid URL")
	}

	// Ensure scheme is present
	if parsedURL.Scheme == "" {
		return fmt.Errorf("URL must include a scheme (http, https, etc.)")
	}

	// Ensure host is present
	if parsedURL.Host == "" {
		return fmt.Errorf("URL must include a host")
	}

	return nil
}

// PhoneValidator validates phone numbers in E.164 format
type PhoneValidator struct{}

// Validate implements the Validator interface
func (v *PhoneValidator) Validate(value interface{}) error {
	if value == nil {
		return nil
	}

	strVal, ok := value.(string)
	if !ok {
		return fmt.Errorf("phone validation requires string value")
	}

	if strings.TrimSpace(strVal) == "" {
		return fmt.Errorf("phone number cannot be empty")
	}

	// E.164 format validation: +[country code][number]
	// Must start with +, followed by 1-15 digits
	if !e164Pattern.MatchString(strVal) {
		return fmt.Errorf("must be a valid phone number in E.164 format (+[country code][number])")
	}

	return nil
}

// MinLengthValidator validates minimum length for arrays
type MinLengthValidator struct {
	MinLength int
}

// Validate implements the Validator interface
func (v *MinLengthValidator) Validate(value interface{}) error {
	if value == nil {
		return nil
	}

	// Use reflection to get length of array/slice
	val := reflect.ValueOf(value)
	if val.Kind() != reflect.Slice && val.Kind() != reflect.Array {
		return fmt.Errorf("min_length validation requires array or slice value")
	}

	if val.Len() < v.MinLength {
		return fmt.Errorf("must contain at least %d items", v.MinLength)
	}

	return nil
}

// MaxLengthValidator validates maximum length for arrays
type MaxLengthValidator struct {
	MaxLength int
}

// Validate implements the Validator interface
func (v *MaxLengthValidator) Validate(value interface{}) error {
	if value == nil {
		return nil
	}

	// Use reflection to get length of array/slice
	val := reflect.ValueOf(value)
	if val.Kind() != reflect.Slice && val.Kind() != reflect.Array {
		return fmt.Errorf("max_length validation requires array or slice value")
	}

	if val.Len() > v.MaxLength {
		return fmt.Errorf("must contain at most %d items", v.MaxLength)
	}

	return nil
}

// Helper functions for type conversion

func toInt64(value interface{}) (int64, bool) {
	switch v := value.(type) {
	case int:
		return int64(v), true
	case int8:
		return int64(v), true
	case int16:
		return int64(v), true
	case int32:
		return int64(v), true
	case int64:
		return v, true
	case uint:
		return int64(v), true
	case uint8:
		return int64(v), true
	case uint16:
		return int64(v), true
	case uint32:
		return int64(v), true
	case uint64:
		return int64(v), true
	case float32:
		return int64(v), true
	case float64:
		return int64(v), true
	default:
		return 0, false
	}
}

func toFloat64(value interface{}) (float64, bool) {
	switch v := value.(type) {
	case int:
		return float64(v), true
	case int8:
		return float64(v), true
	case int16:
		return float64(v), true
	case int32:
		return float64(v), true
	case int64:
		return float64(v), true
	case uint:
		return float64(v), true
	case uint8:
		return float64(v), true
	case uint16:
		return float64(v), true
	case uint32:
		return float64(v), true
	case uint64:
		return float64(v), true
	case float32:
		return float64(v), true
	case float64:
		return v, true
	default:
		return 0, false
	}
}
