package runtime

import (
	"testing"
	"time"
)

// String Namespace Tests

func TestStringLength(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  int
	}{
		{"empty string", "", 0},
		{"ascii string", "hello", 5},
		{"unicode string", "你好世界", 4},
		{"emoji", "👋🌍", 2},
		{"mixed content", "Hello 世界 👋", 10}, // H-e-l-l-o-space-世-界-space-👋 = 10 runes
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := StringLength(tt.input)
			if got != tt.want {
				t.Errorf("StringLength(%q) = %d, want %d", tt.input, got, tt.want)
			}
		})
	}
}

func TestStringSlugify(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"basic text", "Hello World", "hello-world"},
		{"with punctuation", "Hello, World!", "hello-world"},
		{"multiple spaces", "  Multiple   Spaces  ", "multiple-spaces"},
		{"special chars", "C++ Programming!", "c-programming"},
		{"already slug", "already-a-slug", "already-a-slug"},
		{"numbers", "Post 123", "post-123"},
		{"leading/trailing dashes", "---test---", "test"},
		{"unicode chars", "Hello 世界", "hello"},
		{"empty string", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := StringSlugify(tt.input)
			if got != tt.want {
				t.Errorf("StringSlugify(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestStringUpcase(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"lowercase", "hello", "HELLO"},
		{"mixed case", "HeLLo", "HELLO"},
		{"already uppercase", "HELLO", "HELLO"},
		{"with numbers", "hello123", "HELLO123"},
		{"empty string", "", ""},
		{"unicode", "héllo", "HÉLLO"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := StringUpcase(tt.input)
			if got != tt.want {
				t.Errorf("StringUpcase(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestStringDowncase(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"uppercase", "HELLO", "hello"},
		{"mixed case", "HeLLo", "hello"},
		{"already lowercase", "hello", "hello"},
		{"with numbers", "HELLO123", "hello123"},
		{"empty string", "", ""},
		{"unicode", "HÉLLO", "héllo"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := StringDowncase(tt.input)
			if got != tt.want {
				t.Errorf("StringDowncase(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestStringTrim(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"leading spaces", "  hello", "hello"},
		{"trailing spaces", "hello  ", "hello"},
		{"both sides", "  hello  ", "hello"},
		{"tabs and newlines", "\t\nhello\n\t", "hello"},
		{"no whitespace", "hello", "hello"},
		{"only whitespace", "   ", ""},
		{"empty string", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := StringTrim(tt.input)
			if got != tt.want {
				t.Errorf("StringTrim(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestStringContains(t *testing.T) {
	tests := []struct {
		name   string
		str    string
		substr string
		want   bool
	}{
		{"contains at start", "hello world", "hello", true},
		{"contains at end", "hello world", "world", true},
		{"contains in middle", "hello world", "lo wo", true},
		{"does not contain", "hello world", "xyz", false},
		{"empty substring", "hello", "", true},
		{"empty string", "", "hello", false},
		{"both empty", "", "", true},
		{"case sensitive", "Hello", "hello", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := StringContains(tt.str, tt.substr)
			if got != tt.want {
				t.Errorf("StringContains(%q, %q) = %v, want %v", tt.str, tt.substr, got, tt.want)
			}
		})
	}
}

func TestStringReplace(t *testing.T) {
	tests := []struct {
		name string
		str  string
		old  string
		new  string
		want string
	}{
		{"single occurrence", "hello world", "world", "universe", "hello universe"},
		{"multiple occurrences", "foo bar foo", "foo", "baz", "baz bar baz"},
		{"no match", "hello world", "xyz", "abc", "hello world"},
		// Note: Go's strings.ReplaceAll with empty "old" inserts between every rune
		{"empty old", "hello", "", "x", "xhxexlxlxox"},
		{"empty new", "hello world", "world", "", "hello "},
		{"replace with same", "hello", "hello", "hello", "hello"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := StringReplace(tt.str, tt.old, tt.new)
			if got != tt.want {
				t.Errorf("StringReplace(%q, %q, %q) = %q, want %q", tt.str, tt.old, tt.new, got, tt.want)
			}
		})
	}
}

// Time Namespace Tests

func TestTimeNow(t *testing.T) {
	before := time.Now()
	result := TimeNow()
	after := time.Now()

	if result.Before(before) || result.After(after) {
		t.Errorf("TimeNow() returned time outside expected range")
	}
}

func TestTimeFormat(t *testing.T) {
	// Create a fixed time for testing
	testTime := time.Date(2025, 10, 17, 14, 30, 45, 0, time.UTC)

	tests := []struct {
		name   string
		time   time.Time
		layout string
		want   string
	}{
		{"date only", testTime, "2006-01-02", "2025-10-17"},
		{"datetime", testTime, "2006-01-02 15:04:05", "2025-10-17 14:30:45"},
		{"month name", testTime, "Jan 2, 2006", "Oct 17, 2025"},
		{"time only", testTime, "15:04:05", "14:30:45"},
		{"RFC3339", testTime, time.RFC3339, "2025-10-17T14:30:45Z"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := TimeFormat(tt.time, tt.layout)
			if got != tt.want {
				t.Errorf("TimeFormat(%v, %q) = %q, want %q", tt.time, tt.layout, got, tt.want)
			}
		})
	}
}

func TestTimeParse(t *testing.T) {
	tests := []struct {
		name    string
		str     string
		layout  string
		wantNil bool
		want    time.Time
	}{
		{
			name:    "valid date",
			str:     "2025-10-17",
			layout:  "2006-01-02",
			wantNil: false,
			want:    time.Date(2025, 10, 17, 0, 0, 0, 0, time.UTC),
		},
		{
			name:    "valid datetime",
			str:     "2025-10-17 14:30:45",
			layout:  "2006-01-02 15:04:05",
			wantNil: false,
			want:    time.Date(2025, 10, 17, 14, 30, 45, 0, time.UTC),
		},
		{
			name:    "invalid format",
			str:     "not-a-date",
			layout:  "2006-01-02",
			wantNil: true,
		},
		{
			name:    "mismatched layout",
			str:     "2025-10-17",
			layout:  "2006-01-02 15:04:05",
			wantNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := TimeParse(tt.str, tt.layout)
			if tt.wantNil {
				if got != nil {
					t.Errorf("TimeParse(%q, %q) = %v, want nil", tt.str, tt.layout, got)
				}
			} else {
				if got == nil {
					t.Errorf("TimeParse(%q, %q) = nil, want non-nil", tt.str, tt.layout)
				} else if !got.Equal(tt.want) {
					t.Errorf("TimeParse(%q, %q) = %v, want %v", tt.str, tt.layout, got, tt.want)
				}
			}
		})
	}
}

func TestTimeAddDays(t *testing.T) {
	baseTime := time.Date(2025, 10, 17, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name string
		time time.Time
		days int
		want time.Time
	}{
		{
			name: "add positive days",
			time: baseTime,
			days: 5,
			want: time.Date(2025, 10, 22, 12, 0, 0, 0, time.UTC),
		},
		{
			name: "add negative days",
			time: baseTime,
			days: -3,
			want: time.Date(2025, 10, 14, 12, 0, 0, 0, time.UTC),
		},
		{
			name: "add zero days",
			time: baseTime,
			days: 0,
			want: baseTime,
		},
		{
			name: "cross month boundary",
			time: time.Date(2025, 10, 30, 12, 0, 0, 0, time.UTC),
			days: 5,
			want: time.Date(2025, 11, 4, 12, 0, 0, 0, time.UTC),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := TimeAddDays(tt.time, tt.days)
			if !got.Equal(tt.want) {
				t.Errorf("TimeAddDays(%v, %d) = %v, want %v", tt.time, tt.days, got, tt.want)
			}
		})
	}
}

// Array Namespace Tests

func TestArrayLength(t *testing.T) {
	tests := []struct {
		name  string
		input interface{}
		want  int
	}{
		{"empty slice", []interface{}{}, 0},
		{"string slice", []string{"a", "b", "c"}, 3},
		{"int slice", []int{1, 2, 3, 4, 5}, 5},
		{"float slice", []float64{1.1, 2.2}, 2},
		{"bool slice", []bool{true, false, true}, 3},
		{"interface slice", []interface{}{1, "two", 3.0}, 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ArrayLength(tt.input)
			if got != tt.want {
				t.Errorf("ArrayLength(%v) = %d, want %d", tt.input, got, tt.want)
			}
		})
	}
}

func TestArrayContains(t *testing.T) {
	tests := []struct {
		name  string
		arr   interface{}
		value interface{}
		want  bool
	}{
		{"string contains", []string{"a", "b", "c"}, "b", true},
		{"string not contains", []string{"a", "b", "c"}, "d", false},
		{"int contains", []int{1, 2, 3}, 2, true},
		{"int not contains", []int{1, 2, 3}, 4, false},
		{"float contains", []float64{1.1, 2.2, 3.3}, 2.2, true},
		{"bool contains", []bool{true, false}, false, true},
		{"empty array", []string{}, "a", false},
		{"interface contains", []interface{}{1, "two", 3.0}, "two", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ArrayContains(tt.arr, tt.value)
			if got != tt.want {
				t.Errorf("ArrayContains(%v, %v) = %v, want %v", tt.arr, tt.value, got, tt.want)
			}
		})
	}
}

// Hash Namespace Tests

func TestHashHasKey(t *testing.T) {
	tests := []struct {
		name string
		hash interface{}
		key  interface{}
		want bool
	}{
		{
			name: "string key exists",
			hash: map[string]interface{}{"a": 1, "b": 2},
			key:  "a",
			want: true,
		},
		{
			name: "string key not exists",
			hash: map[string]interface{}{"a": 1, "b": 2},
			key:  "c",
			want: false,
		},
		{
			name: "string to string map",
			hash: map[string]string{"name": "John", "city": "NYC"},
			key:  "name",
			want: true,
		},
		{
			name: "string to int map",
			hash: map[string]int{"one": 1, "two": 2},
			key:  "one",
			want: true,
		},
		{
			name: "int key exists",
			hash: map[int]interface{}{1: "a", 2: "b"},
			key:  1,
			want: true,
		},
		{
			name: "int key not exists",
			hash: map[int]interface{}{1: "a", 2: "b"},
			key:  3,
			want: false,
		},
		{
			name: "empty map",
			hash: map[string]interface{}{},
			key:  "a",
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := HashHasKey(tt.hash, tt.key)
			if got != tt.want {
				t.Errorf("HashHasKey(%v, %v) = %v, want %v", tt.hash, tt.key, got, tt.want)
			}
		})
	}
}

// UUID Namespace Tests

func TestUUIDGenerate(t *testing.T) {
	// Test that UUIDGenerate returns a valid UUID
	uuid1 := UUIDGenerate()
	uuid2 := UUIDGenerate()

	// Check that UUIDs are not empty
	if uuid1 == "" {
		t.Error("UUIDGenerate() returned empty string")
	}

	// Check that UUIDs are different
	if uuid1 == uuid2 {
		t.Error("UUIDGenerate() returned same UUID twice")
	}

	// Check that UUIDs have correct length (36 characters with hyphens)
	if len(uuid1) != 36 {
		t.Errorf("UUIDGenerate() returned UUID with length %d, want 36", len(uuid1))
	}

	// Check that UUIDs have hyphens in correct positions
	if uuid1[8] != '-' || uuid1[13] != '-' || uuid1[18] != '-' || uuid1[23] != '-' {
		t.Errorf("UUIDGenerate() returned UUID with incorrect format: %s", uuid1)
	}
}

// Benchmark tests

func BenchmarkStringLength(b *testing.B) {
	s := "Hello, 世界!"
	for i := 0; i < b.N; i++ {
		StringLength(s)
	}
}

func BenchmarkStringSlugify(b *testing.B) {
	s := "Hello World! This is a test."
	for i := 0; i < b.N; i++ {
		StringSlugify(s)
	}
}

func BenchmarkArrayContains(b *testing.B) {
	arr := []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
	value := 7
	for i := 0; i < b.N; i++ {
		ArrayContains(arr, value)
	}
}

func BenchmarkUUIDGenerate(b *testing.B) {
	for i := 0; i < b.N; i++ {
		UUIDGenerate()
	}
}
