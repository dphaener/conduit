package validation

import (
	"regexp"
	"testing"

	"github.com/conduit-lang/conduit/internal/orm/schema"
)

func TestMinValidator_Numeric(t *testing.T) {
	tests := []struct {
		name      string
		validator *MinValidator
		value     interface{}
		wantErr   bool
	}{
		{
			name:      "int value above min",
			validator: &MinValidator{Min: 5, FieldType: schema.TypeInt},
			value:     10,
			wantErr:   false,
		},
		{
			name:      "int value below min",
			validator: &MinValidator{Min: 5, FieldType: schema.TypeInt},
			value:     3,
			wantErr:   true,
		},
		{
			name:      "int value equal to min",
			validator: &MinValidator{Min: 5, FieldType: schema.TypeInt},
			value:     5,
			wantErr:   false,
		},
		{
			name:      "float value above min",
			validator: &MinValidator{Min: 5.5, FieldType: schema.TypeFloat},
			value:     10.2,
			wantErr:   false,
		},
		{
			name:      "float value below min",
			validator: &MinValidator{Min: 5.5, FieldType: schema.TypeFloat},
			value:     3.2,
			wantErr:   true,
		},
		{
			name:      "nil value",
			validator: &MinValidator{Min: 5, FieldType: schema.TypeInt},
			value:     nil,
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.validator.Validate(tt.value)
			if (err != nil) != tt.wantErr {
				t.Errorf("MinValidator.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestMinValidator_String(t *testing.T) {
	tests := []struct {
		name      string
		validator *MinValidator
		value     interface{}
		wantErr   bool
	}{
		{
			name:      "string longer than min",
			validator: &MinValidator{Min: 5, FieldType: schema.TypeString},
			value:     "hello world",
			wantErr:   false,
		},
		{
			name:      "string shorter than min",
			validator: &MinValidator{Min: 5, FieldType: schema.TypeString},
			value:     "hi",
			wantErr:   true,
		},
		{
			name:      "string equal to min",
			validator: &MinValidator{Min: 5, FieldType: schema.TypeString},
			value:     "hello",
			wantErr:   false,
		},
		{
			name:      "empty string",
			validator: &MinValidator{Min: 1, FieldType: schema.TypeString},
			value:     "",
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.validator.Validate(tt.value)
			if (err != nil) != tt.wantErr {
				t.Errorf("MinValidator.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestMaxValidator_Numeric(t *testing.T) {
	tests := []struct {
		name      string
		validator *MaxValidator
		value     interface{}
		wantErr   bool
	}{
		{
			name:      "int value below max",
			validator: &MaxValidator{Max: 100, FieldType: schema.TypeInt},
			value:     50,
			wantErr:   false,
		},
		{
			name:      "int value above max",
			validator: &MaxValidator{Max: 100, FieldType: schema.TypeInt},
			value:     150,
			wantErr:   true,
		},
		{
			name:      "int value equal to max",
			validator: &MaxValidator{Max: 100, FieldType: schema.TypeInt},
			value:     100,
			wantErr:   false,
		},
		{
			name:      "float value below max",
			validator: &MaxValidator{Max: 100.5, FieldType: schema.TypeFloat},
			value:     50.2,
			wantErr:   false,
		},
		{
			name:      "float value above max",
			validator: &MaxValidator{Max: 100.5, FieldType: schema.TypeFloat},
			value:     150.7,
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.validator.Validate(tt.value)
			if (err != nil) != tt.wantErr {
				t.Errorf("MaxValidator.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestMaxValidator_String(t *testing.T) {
	tests := []struct {
		name      string
		validator *MaxValidator
		value     interface{}
		wantErr   bool
	}{
		{
			name:      "string shorter than max",
			validator: &MaxValidator{Max: 10, FieldType: schema.TypeString},
			value:     "hello",
			wantErr:   false,
		},
		{
			name:      "string longer than max",
			validator: &MaxValidator{Max: 10, FieldType: schema.TypeString},
			value:     "hello world this is too long",
			wantErr:   true,
		},
		{
			name:      "string equal to max",
			validator: &MaxValidator{Max: 5, FieldType: schema.TypeString},
			value:     "hello",
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.validator.Validate(tt.value)
			if (err != nil) != tt.wantErr {
				t.Errorf("MaxValidator.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestPatternValidator(t *testing.T) {
	tests := []struct {
		name      string
		pattern   string
		value     interface{}
		wantErr   bool
	}{
		{
			name:    "matching pattern",
			pattern: `^[a-z]+$`,
			value:   "hello",
			wantErr: false,
		},
		{
			name:    "non-matching pattern",
			pattern: `^[a-z]+$`,
			value:   "Hello123",
			wantErr: true,
		},
		{
			name:    "email pattern matching",
			pattern: `^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`,
			value:   "user@example.com",
			wantErr: false,
		},
		{
			name:    "email pattern non-matching",
			pattern: `^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`,
			value:   "invalid-email",
			wantErr: true,
		},
		{
			name:    "nil value",
			pattern: `^[a-z]+$`,
			value:   nil,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pattern := regexp.MustCompile(tt.pattern)
			validator := &PatternValidator{Pattern: pattern}
			err := validator.Validate(tt.value)
			if (err != nil) != tt.wantErr {
				t.Errorf("PatternValidator.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestEmailValidator(t *testing.T) {
	tests := []struct {
		name    string
		value   interface{}
		wantErr bool
	}{
		{
			name:    "valid email",
			value:   "user@example.com",
			wantErr: false,
		},
		{
			name:    "valid email with subdomain",
			value:   "user@mail.example.com",
			wantErr: false,
		},
		{
			name:    "valid email with plus",
			value:   "user+tag@example.com",
			wantErr: false,
		},
		{
			name:    "invalid email - no @",
			value:   "userexample.com",
			wantErr: true,
		},
		{
			name:    "invalid email - no domain",
			value:   "user@",
			wantErr: true,
		},
		{
			name:    "invalid email - no user",
			value:   "@example.com",
			wantErr: true,
		},
		{
			name:    "empty string",
			value:   "",
			wantErr: true,
		},
		{
			name:    "whitespace only",
			value:   "   ",
			wantErr: true,
		},
		{
			name:    "nil value",
			value:   nil,
			wantErr: false,
		},
		{
			name:    "non-string value",
			value:   123,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validator := &EmailValidator{}
			err := validator.Validate(tt.value)
			if (err != nil) != tt.wantErr {
				t.Errorf("EmailValidator.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestURLValidator(t *testing.T) {
	tests := []struct {
		name    string
		value   interface{}
		wantErr bool
	}{
		{
			name:    "valid http URL",
			value:   "http://example.com",
			wantErr: false,
		},
		{
			name:    "valid https URL",
			value:   "https://example.com/path",
			wantErr: false,
		},
		{
			name:    "valid URL with port",
			value:   "http://example.com:8080",
			wantErr: false,
		},
		{
			name:    "valid URL with query",
			value:   "https://example.com/path?key=value",
			wantErr: false,
		},
		{
			name:    "invalid URL - no scheme",
			value:   "example.com",
			wantErr: true,
		},
		{
			name:    "invalid URL - no host",
			value:   "http://",
			wantErr: true,
		},
		{
			name:    "empty string",
			value:   "",
			wantErr: true,
		},
		{
			name:    "whitespace only",
			value:   "   ",
			wantErr: true,
		},
		{
			name:    "nil value",
			value:   nil,
			wantErr: false,
		},
		{
			name:    "non-string value",
			value:   123,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validator := &URLValidator{}
			err := validator.Validate(tt.value)
			if (err != nil) != tt.wantErr {
				t.Errorf("URLValidator.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestMinLengthValidator(t *testing.T) {
	tests := []struct {
		name      string
		validator *MinLengthValidator
		value     interface{}
		wantErr   bool
	}{
		{
			name:      "array longer than min",
			validator: &MinLengthValidator{MinLength: 2},
			value:     []string{"a", "b", "c"},
			wantErr:   false,
		},
		{
			name:      "array shorter than min",
			validator: &MinLengthValidator{MinLength: 5},
			value:     []string{"a", "b"},
			wantErr:   true,
		},
		{
			name:      "array equal to min",
			validator: &MinLengthValidator{MinLength: 3},
			value:     []int{1, 2, 3},
			wantErr:   false,
		},
		{
			name:      "empty array",
			validator: &MinLengthValidator{MinLength: 1},
			value:     []string{},
			wantErr:   true,
		},
		{
			name:      "nil value",
			validator: &MinLengthValidator{MinLength: 1},
			value:     nil,
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.validator.Validate(tt.value)
			if (err != nil) != tt.wantErr {
				t.Errorf("MinLengthValidator.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestMaxLengthValidator(t *testing.T) {
	tests := []struct {
		name      string
		validator *MaxLengthValidator
		value     interface{}
		wantErr   bool
	}{
		{
			name:      "array shorter than max",
			validator: &MaxLengthValidator{MaxLength: 5},
			value:     []string{"a", "b"},
			wantErr:   false,
		},
		{
			name:      "array longer than max",
			validator: &MaxLengthValidator{MaxLength: 2},
			value:     []string{"a", "b", "c"},
			wantErr:   true,
		},
		{
			name:      "array equal to max",
			validator: &MaxLengthValidator{MaxLength: 3},
			value:     []int{1, 2, 3},
			wantErr:   false,
		},
		{
			name:      "empty array",
			validator: &MaxLengthValidator{MaxLength: 5},
			value:     []string{},
			wantErr:   false,
		},
		{
			name:      "nil value",
			validator: &MaxLengthValidator{MaxLength: 5},
			value:     nil,
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.validator.Validate(tt.value)
			if (err != nil) != tt.wantErr {
				t.Errorf("MaxLengthValidator.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestToInt64(t *testing.T) {
	tests := []struct {
		name     string
		value    interface{}
		expected int64
		ok       bool
	}{
		{"int", 42, 42, true},
		{"int8", int8(42), 42, true},
		{"int16", int16(42), 42, true},
		{"int32", int32(42), 42, true},
		{"int64", int64(42), 42, true},
		{"uint", uint(42), 42, true},
		{"float64", float64(42.0), 42, true},
		{"string", "42", 0, false},
		{"nil", nil, 0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, ok := toInt64(tt.value)
			if ok != tt.ok {
				t.Errorf("toInt64(%v) ok = %v, want %v", tt.value, ok, tt.ok)
			}
			if ok && result != tt.expected {
				t.Errorf("toInt64(%v) = %v, want %v", tt.value, result, tt.expected)
			}
		})
	}
}

func TestToFloat64(t *testing.T) {
	tests := []struct {
		name     string
		value    interface{}
		expected float64
		ok       bool
	}{
		{"int", 42, 42.0, true},
		{"int64", int64(42), 42.0, true},
		{"float32", float32(42.5), 42.5, true},
		{"float64", float64(42.5), 42.5, true},
		{"string", "42.5", 0, false},
		{"nil", nil, 0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, ok := toFloat64(tt.value)
			if ok != tt.ok {
				t.Errorf("toFloat64(%v) ok = %v, want %v", tt.value, ok, tt.ok)
			}
			if ok && result != tt.expected {
				t.Errorf("toFloat64(%v) = %v, want %v", tt.value, result, tt.expected)
			}
		})
	}
}
