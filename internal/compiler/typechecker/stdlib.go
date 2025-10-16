package typechecker

// FunctionParam represents a parameter in a function signature
type FunctionParam struct {
	Name     string
	Type     Type
	Optional bool // For named parameters with defaults
}

// Function represents a standard library or custom function signature
type Function struct {
	Name       string
	Namespace  string // Empty for custom functions
	Parameters []FunctionParam
	ReturnType Type
}

// FullName returns the fully qualified function name (Namespace.Name or just Name)
func (f *Function) FullName() string {
	if f.Namespace != "" {
		return f.Namespace + "." + f.Name
	}
	return f.Name
}

// StdlibFunctions contains all standard library function signatures
var StdlibFunctions = map[string]map[string]*Function{
	"String": {
		"slugify": {
			Name:      "slugify",
			Namespace: "String",
			Parameters: []FunctionParam{
				{Name: "text", Type: NewPrimitiveType("string", false)},
			},
			ReturnType: NewPrimitiveType("string", false),
		},
		"capitalize": {
			Name:      "capitalize",
			Namespace: "String",
			Parameters: []FunctionParam{
				{Name: "text", Type: NewPrimitiveType("string", false)},
			},
			ReturnType: NewPrimitiveType("string", false),
		},
		"upcase": {
			Name:      "upcase",
			Namespace: "String",
			Parameters: []FunctionParam{
				{Name: "text", Type: NewPrimitiveType("string", false)},
			},
			ReturnType: NewPrimitiveType("string", false),
		},
		"downcase": {
			Name:      "downcase",
			Namespace: "String",
			Parameters: []FunctionParam{
				{Name: "text", Type: NewPrimitiveType("string", false)},
			},
			ReturnType: NewPrimitiveType("string", false),
		},
		"trim": {
			Name:      "trim",
			Namespace: "String",
			Parameters: []FunctionParam{
				{Name: "text", Type: NewPrimitiveType("string", false)},
			},
			ReturnType: NewPrimitiveType("string", false),
		},
		"truncate": {
			Name:      "truncate",
			Namespace: "String",
			Parameters: []FunctionParam{
				{Name: "text", Type: NewPrimitiveType("string", false)},
				{Name: "length", Type: NewPrimitiveType("int", false)},
			},
			ReturnType: NewPrimitiveType("string", false),
		},
		"split": {
			Name:      "split",
			Namespace: "String",
			Parameters: []FunctionParam{
				{Name: "text", Type: NewPrimitiveType("string", false)},
				{Name: "delimiter", Type: NewPrimitiveType("string", false)},
			},
			ReturnType: NewArrayType(NewPrimitiveType("string", false), false),
		},
		"join": {
			Name:      "join",
			Namespace: "String",
			Parameters: []FunctionParam{
				{Name: "parts", Type: NewArrayType(NewPrimitiveType("string", false), false)},
				{Name: "delimiter", Type: NewPrimitiveType("string", false)},
			},
			ReturnType: NewPrimitiveType("string", false),
		},
		"replace": {
			Name:      "replace",
			Namespace: "String",
			Parameters: []FunctionParam{
				{Name: "text", Type: NewPrimitiveType("string", false)},
				{Name: "pattern", Type: NewPrimitiveType("string", false)},
				{Name: "replacement", Type: NewPrimitiveType("string", false)},
			},
			ReturnType: NewPrimitiveType("string", false),
		},
		"starts_with?": {
			Name:      "starts_with?",
			Namespace: "String",
			Parameters: []FunctionParam{
				{Name: "text", Type: NewPrimitiveType("string", false)},
				{Name: "prefix", Type: NewPrimitiveType("string", false)},
			},
			ReturnType: NewPrimitiveType("bool", false),
		},
		"ends_with?": {
			Name:      "ends_with?",
			Namespace: "String",
			Parameters: []FunctionParam{
				{Name: "text", Type: NewPrimitiveType("string", false)},
				{Name: "suffix", Type: NewPrimitiveType("string", false)},
			},
			ReturnType: NewPrimitiveType("bool", false),
		},
		"includes?": {
			Name:      "includes?",
			Namespace: "String",
			Parameters: []FunctionParam{
				{Name: "text", Type: NewPrimitiveType("string", false)},
				{Name: "substring", Type: NewPrimitiveType("string", false)},
			},
			ReturnType: NewPrimitiveType("bool", false),
		},
		"length": {
			Name:      "length",
			Namespace: "String",
			Parameters: []FunctionParam{
				{Name: "text", Type: NewPrimitiveType("string", false)},
			},
			ReturnType: NewPrimitiveType("int", false),
		},
	},
	"Text": {
		"calculate_reading_time": {
			Name:      "calculate_reading_time",
			Namespace: "Text",
			Parameters: []FunctionParam{
				{Name: "content", Type: NewPrimitiveType("text", false)},
				{Name: "words_per_minute", Type: NewPrimitiveType("int", false)},
			},
			ReturnType: NewPrimitiveType("int", false),
		},
		"word_count": {
			Name:      "word_count",
			Namespace: "Text",
			Parameters: []FunctionParam{
				{Name: "content", Type: NewPrimitiveType("text", false)},
			},
			ReturnType: NewPrimitiveType("int", false),
		},
		"character_count": {
			Name:      "character_count",
			Namespace: "Text",
			Parameters: []FunctionParam{
				{Name: "content", Type: NewPrimitiveType("text", false)},
			},
			ReturnType: NewPrimitiveType("int", false),
		},
		"excerpt": {
			Name:      "excerpt",
			Namespace: "Text",
			Parameters: []FunctionParam{
				{Name: "content", Type: NewPrimitiveType("text", false)},
				{Name: "length", Type: NewPrimitiveType("int", false)},
			},
			ReturnType: NewPrimitiveType("string", false),
		},
	},
	"Number": {
		"format": {
			Name:      "format",
			Namespace: "Number",
			Parameters: []FunctionParam{
				{Name: "num", Type: NewPrimitiveType("float", false)},
				{Name: "decimals", Type: NewPrimitiveType("int", false)},
			},
			ReturnType: NewPrimitiveType("string", false),
		},
		"round": {
			Name:      "round",
			Namespace: "Number",
			Parameters: []FunctionParam{
				{Name: "num", Type: NewPrimitiveType("float", false)},
				{Name: "precision", Type: NewPrimitiveType("int", false)},
			},
			ReturnType: NewPrimitiveType("float", false),
		},
		"abs": {
			Name:      "abs",
			Namespace: "Number",
			Parameters: []FunctionParam{
				{Name: "num", Type: NewPrimitiveType("float", false)},
			},
			ReturnType: NewPrimitiveType("float", false),
		},
		"ceil": {
			Name:      "ceil",
			Namespace: "Number",
			Parameters: []FunctionParam{
				{Name: "num", Type: NewPrimitiveType("float", false)},
			},
			ReturnType: NewPrimitiveType("int", false),
		},
		"floor": {
			Name:      "floor",
			Namespace: "Number",
			Parameters: []FunctionParam{
				{Name: "num", Type: NewPrimitiveType("float", false)},
			},
			ReturnType: NewPrimitiveType("int", false),
		},
		"min": {
			Name:      "min",
			Namespace: "Number",
			Parameters: []FunctionParam{
				{Name: "a", Type: NewPrimitiveType("float", false)},
				{Name: "b", Type: NewPrimitiveType("float", false)},
			},
			ReturnType: NewPrimitiveType("float", false),
		},
		"max": {
			Name:      "max",
			Namespace: "Number",
			Parameters: []FunctionParam{
				{Name: "a", Type: NewPrimitiveType("float", false)},
				{Name: "b", Type: NewPrimitiveType("float", false)},
			},
			ReturnType: NewPrimitiveType("float", false),
		},
	},
	"Array": {
		"first": {
			Name:      "first",
			Namespace: "Array",
			Parameters: []FunctionParam{
				{Name: "arr", Type: NewArrayType(NewPrimitiveType("any", false), false)},
			},
			ReturnType: NewPrimitiveType("any", true), // Returns nullable
		},
		"last": {
			Name:      "last",
			Namespace: "Array",
			Parameters: []FunctionParam{
				{Name: "arr", Type: NewArrayType(NewPrimitiveType("any", false), false)},
			},
			ReturnType: NewPrimitiveType("any", true), // Returns nullable
		},
		"length": {
			Name:      "length",
			Namespace: "Array",
			Parameters: []FunctionParam{
				{Name: "arr", Type: NewArrayType(NewPrimitiveType("any", false), false)},
			},
			ReturnType: NewPrimitiveType("int", false),
		},
		"empty?": {
			Name:      "empty?",
			Namespace: "Array",
			Parameters: []FunctionParam{
				{Name: "arr", Type: NewArrayType(NewPrimitiveType("any", false), false)},
			},
			ReturnType: NewPrimitiveType("bool", false),
		},
		"includes?": {
			Name:      "includes?",
			Namespace: "Array",
			Parameters: []FunctionParam{
				{Name: "arr", Type: NewArrayType(NewPrimitiveType("any", false), false)},
				{Name: "item", Type: NewPrimitiveType("any", false)},
			},
			ReturnType: NewPrimitiveType("bool", false),
		},
		"unique": {
			Name:      "unique",
			Namespace: "Array",
			Parameters: []FunctionParam{
				{Name: "arr", Type: NewArrayType(NewPrimitiveType("any", false), false)},
			},
			ReturnType: NewArrayType(NewPrimitiveType("any", false), false),
		},
		"sort": {
			Name:      "sort",
			Namespace: "Array",
			Parameters: []FunctionParam{
				{Name: "arr", Type: NewArrayType(NewPrimitiveType("any", false), false)},
			},
			ReturnType: NewArrayType(NewPrimitiveType("any", false), false),
		},
		"reverse": {
			Name:      "reverse",
			Namespace: "Array",
			Parameters: []FunctionParam{
				{Name: "arr", Type: NewArrayType(NewPrimitiveType("any", false), false)},
			},
			ReturnType: NewArrayType(NewPrimitiveType("any", false), false),
		},
		"push": {
			Name:      "push",
			Namespace: "Array",
			Parameters: []FunctionParam{
				{Name: "arr", Type: NewArrayType(NewPrimitiveType("any", false), false)},
				{Name: "item", Type: NewPrimitiveType("any", false)},
			},
			ReturnType: NewArrayType(NewPrimitiveType("any", false), false),
		},
		"concat": {
			Name:      "concat",
			Namespace: "Array",
			Parameters: []FunctionParam{
				{Name: "arr1", Type: NewArrayType(NewPrimitiveType("any", false), false)},
				{Name: "arr2", Type: NewArrayType(NewPrimitiveType("any", false), false)},
			},
			ReturnType: NewArrayType(NewPrimitiveType("any", false), false),
		},
		"map": {
			Name:      "map",
			Namespace: "Array",
			Parameters: []FunctionParam{
				{Name: "arr", Type: NewArrayType(NewPrimitiveType("any", false), false)},
				{Name: "fn", Type: NewPrimitiveType("any", false)},
			},
			ReturnType: NewArrayType(NewPrimitiveType("any", false), false),
		},
		"filter": {
			Name:      "filter",
			Namespace: "Array",
			Parameters: []FunctionParam{
				{Name: "arr", Type: NewArrayType(NewPrimitiveType("any", false), false)},
				{Name: "fn", Type: NewPrimitiveType("any", false)},
			},
			ReturnType: NewArrayType(NewPrimitiveType("any", false), false),
		},
		"reduce": {
			Name:      "reduce",
			Namespace: "Array",
			Parameters: []FunctionParam{
				{Name: "arr", Type: NewArrayType(NewPrimitiveType("any", false), false)},
				{Name: "initial", Type: NewPrimitiveType("any", false)},
				{Name: "fn", Type: NewPrimitiveType("any", false)},
			},
			ReturnType: NewPrimitiveType("any", false),
		},
		"count": {
			Name:      "count",
			Namespace: "Array",
			Parameters: []FunctionParam{
				{Name: "arr", Type: NewArrayType(NewPrimitiveType("any", false), false)},
			},
			ReturnType: NewPrimitiveType("int", false),
		},
		"contains": {
			Name:      "contains",
			Namespace: "Array",
			Parameters: []FunctionParam{
				{Name: "arr", Type: NewArrayType(NewPrimitiveType("any", false), false)},
				{Name: "item", Type: NewPrimitiveType("any", false)},
			},
			ReturnType: NewPrimitiveType("bool", false),
		},
	},
	"Hash": {
		"keys": {
			Name:      "keys",
			Namespace: "Hash",
			Parameters: []FunctionParam{
				{Name: "hash", Type: NewHashType(NewPrimitiveType("any", false), NewPrimitiveType("any", false), false)},
			},
			ReturnType: NewArrayType(NewPrimitiveType("any", false), false),
		},
		"values": {
			Name:      "values",
			Namespace: "Hash",
			Parameters: []FunctionParam{
				{Name: "hash", Type: NewHashType(NewPrimitiveType("any", false), NewPrimitiveType("any", false), false)},
			},
			ReturnType: NewArrayType(NewPrimitiveType("any", false), false),
		},
		"merge": {
			Name:      "merge",
			Namespace: "Hash",
			Parameters: []FunctionParam{
				{Name: "hash1", Type: NewHashType(NewPrimitiveType("any", false), NewPrimitiveType("any", false), false)},
				{Name: "hash2", Type: NewHashType(NewPrimitiveType("any", false), NewPrimitiveType("any", false), false)},
			},
			ReturnType: NewHashType(NewPrimitiveType("any", false), NewPrimitiveType("any", false), false),
		},
		"has_key?": {
			Name:      "has_key?",
			Namespace: "Hash",
			Parameters: []FunctionParam{
				{Name: "hash", Type: NewHashType(NewPrimitiveType("any", false), NewPrimitiveType("any", false), false)},
				{Name: "key", Type: NewPrimitiveType("any", false)},
			},
			ReturnType: NewPrimitiveType("bool", false),
		},
		"get": {
			Name:      "get",
			Namespace: "Hash",
			Parameters: []FunctionParam{
				{Name: "hash", Type: NewHashType(NewPrimitiveType("any", false), NewPrimitiveType("any", false), false)},
				{Name: "key", Type: NewPrimitiveType("any", false)},
				{Name: "default", Type: NewPrimitiveType("any", true), Optional: true},
			},
			ReturnType: NewPrimitiveType("any", true),
		},
	},
	"Time": {
		"now": {
			Name:       "now",
			Namespace:  "Time",
			Parameters: []FunctionParam{},
			ReturnType: NewPrimitiveType("timestamp", false),
		},
		"today": {
			Name:       "today",
			Namespace:  "Time",
			Parameters: []FunctionParam{},
			ReturnType: NewPrimitiveType("date", false),
		},
		"parse": {
			Name:      "parse",
			Namespace: "Time",
			Parameters: []FunctionParam{
				{Name: "str", Type: NewPrimitiveType("string", false)},
				{Name: "format", Type: NewPrimitiveType("string", true), Optional: true},
			},
			ReturnType: NewPrimitiveType("timestamp", false),
		},
		"format": {
			Name:      "format",
			Namespace: "Time",
			Parameters: []FunctionParam{
				{Name: "time", Type: NewPrimitiveType("timestamp", false)},
				{Name: "format", Type: NewPrimitiveType("string", false)},
			},
			ReturnType: NewPrimitiveType("string", false),
		},
		"year": {
			Name:      "year",
			Namespace: "Time",
			Parameters: []FunctionParam{
				{Name: "time", Type: NewPrimitiveType("timestamp", false)},
			},
			ReturnType: NewPrimitiveType("int", false),
		},
		"month": {
			Name:      "month",
			Namespace: "Time",
			Parameters: []FunctionParam{
				{Name: "time", Type: NewPrimitiveType("timestamp", false)},
			},
			ReturnType: NewPrimitiveType("int", false),
		},
		"day": {
			Name:      "day",
			Namespace: "Time",
			Parameters: []FunctionParam{
				{Name: "time", Type: NewPrimitiveType("timestamp", false)},
			},
			ReturnType: NewPrimitiveType("int", false),
		},
	},
	"UUID": {
		"generate": {
			Name:       "generate",
			Namespace:  "UUID",
			Parameters: []FunctionParam{},
			ReturnType: NewPrimitiveType("uuid", false),
		},
		"validate": {
			Name:      "validate",
			Namespace: "UUID",
			Parameters: []FunctionParam{
				{Name: "str", Type: NewPrimitiveType("string", false)},
			},
			ReturnType: NewPrimitiveType("bool", false),
		},
		"parse": {
			Name:      "parse",
			Namespace: "UUID",
			Parameters: []FunctionParam{
				{Name: "str", Type: NewPrimitiveType("string", false)},
			},
			ReturnType: NewPrimitiveType("uuid", true),
		},
	},
	"Random": {
		"int": {
			Name:      "int",
			Namespace: "Random",
			Parameters: []FunctionParam{
				{Name: "min", Type: NewPrimitiveType("int", false)},
				{Name: "max", Type: NewPrimitiveType("int", false)},
			},
			ReturnType: NewPrimitiveType("int", false),
		},
		"float": {
			Name:      "float",
			Namespace: "Random",
			Parameters: []FunctionParam{
				{Name: "min", Type: NewPrimitiveType("float", false)},
				{Name: "max", Type: NewPrimitiveType("float", false)},
			},
			ReturnType: NewPrimitiveType("float", false),
		},
		"uuid": {
			Name:       "uuid",
			Namespace:  "Random",
			Parameters: []FunctionParam{},
			ReturnType: NewPrimitiveType("uuid", false),
		},
		"hex": {
			Name:      "hex",
			Namespace: "Random",
			Parameters: []FunctionParam{
				{Name: "length", Type: NewPrimitiveType("int", false)},
			},
			ReturnType: NewPrimitiveType("string", false),
		},
		"alphanumeric": {
			Name:      "alphanumeric",
			Namespace: "Random",
			Parameters: []FunctionParam{
				{Name: "length", Type: NewPrimitiveType("int", false)},
			},
			ReturnType: NewPrimitiveType("string", false),
		},
	},
	"Crypto": {
		"hash": {
			Name:      "hash",
			Namespace: "Crypto",
			Parameters: []FunctionParam{
				{Name: "data", Type: NewPrimitiveType("string", false)},
				{Name: "algorithm", Type: NewPrimitiveType("string", false)},
			},
			ReturnType: NewPrimitiveType("string", false),
		},
		"compare": {
			Name:      "compare",
			Namespace: "Crypto",
			Parameters: []FunctionParam{
				{Name: "hash", Type: NewPrimitiveType("string", false)},
				{Name: "data", Type: NewPrimitiveType("string", false)},
			},
			ReturnType: NewPrimitiveType("bool", false),
		},
	},
	"HTML": {
		"strip_tags": {
			Name:      "strip_tags",
			Namespace: "HTML",
			Parameters: []FunctionParam{
				{Name: "html", Type: NewPrimitiveType("string", false)},
			},
			ReturnType: NewPrimitiveType("string", false),
		},
		"escape": {
			Name:      "escape",
			Namespace: "HTML",
			Parameters: []FunctionParam{
				{Name: "str", Type: NewPrimitiveType("string", false)},
			},
			ReturnType: NewPrimitiveType("string", false),
		},
		"unescape": {
			Name:      "unescape",
			Namespace: "HTML",
			Parameters: []FunctionParam{
				{Name: "str", Type: NewPrimitiveType("string", false)},
			},
			ReturnType: NewPrimitiveType("string", false),
		},
	},
	"JSON": {
		"parse": {
			Name:      "parse",
			Namespace: "JSON",
			Parameters: []FunctionParam{
				{Name: "str", Type: NewPrimitiveType("string", false)},
			},
			ReturnType: NewPrimitiveType("json", false),
		},
		"stringify": {
			Name:      "stringify",
			Namespace: "JSON",
			Parameters: []FunctionParam{
				{Name: "data", Type: NewPrimitiveType("json", false)},
				{Name: "pretty", Type: NewPrimitiveType("bool", true), Optional: true},
			},
			ReturnType: NewPrimitiveType("string", false),
		},
		"validate": {
			Name:      "validate",
			Namespace: "JSON",
			Parameters: []FunctionParam{
				{Name: "str", Type: NewPrimitiveType("string", false)},
			},
			ReturnType: NewPrimitiveType("bool", false),
		},
	},
	"Regex": {
		"match": {
			Name:      "match",
			Namespace: "Regex",
			Parameters: []FunctionParam{
				{Name: "text", Type: NewPrimitiveType("string", false)},
				{Name: "pattern", Type: NewPrimitiveType("string", false)},
			},
			ReturnType: NewArrayType(NewPrimitiveType("string", false), true),
		},
		"replace": {
			Name:      "replace",
			Namespace: "Regex",
			Parameters: []FunctionParam{
				{Name: "text", Type: NewPrimitiveType("string", false)},
				{Name: "pattern", Type: NewPrimitiveType("string", false)},
				{Name: "replacement", Type: NewPrimitiveType("string", false)},
			},
			ReturnType: NewPrimitiveType("string", false),
		},
		"test": {
			Name:      "test",
			Namespace: "Regex",
			Parameters: []FunctionParam{
				{Name: "text", Type: NewPrimitiveType("string", false)},
				{Name: "pattern", Type: NewPrimitiveType("string", false)},
			},
			ReturnType: NewPrimitiveType("bool", false),
		},
		"split": {
			Name:      "split",
			Namespace: "Regex",
			Parameters: []FunctionParam{
				{Name: "text", Type: NewPrimitiveType("string", false)},
				{Name: "pattern", Type: NewPrimitiveType("string", false)},
			},
			ReturnType: NewArrayType(NewPrimitiveType("string", false), false),
		},
	},
	"Logger": {
		"warn": {
			Name:      "warn",
			Namespace: "Logger",
			Parameters: []FunctionParam{
				{Name: "message", Type: NewPrimitiveType("string", false)},
			},
			ReturnType: NewPrimitiveType("void", false),
		},
		"debug": {
			Name:      "debug",
			Namespace: "Logger",
			Parameters: []FunctionParam{
				{Name: "message", Type: NewPrimitiveType("string", false)},
			},
			ReturnType: NewPrimitiveType("void", false),
		},
	},
	"Context": {
		"current_user": {
			Name:       "current_user",
			Namespace:  "Context",
			Parameters: []FunctionParam{},
			ReturnType: NewResourceType("User", true), // Returns nullable User?
		},
		"authenticated?": {
			Name:       "authenticated?",
			Namespace:  "Context",
			Parameters: []FunctionParam{},
			ReturnType: NewPrimitiveType("bool", false),
		},
		"headers": {
			Name:       "headers",
			Namespace:  "Context",
			Parameters: []FunctionParam{},
			ReturnType: NewHashType(NewPrimitiveType("string", false), NewPrimitiveType("string", false), true),
		},
		"request_id": {
			Name:       "request_id",
			Namespace:  "Context",
			Parameters: []FunctionParam{},
			ReturnType: NewPrimitiveType("string", false),
		},
	},
	"Env": {
		"get": {
			Name:      "get",
			Namespace: "Env",
			Parameters: []FunctionParam{
				{Name: "key", Type: NewPrimitiveType("string", false)},
				{Name: "default", Type: NewPrimitiveType("string", true), Optional: true},
			},
			ReturnType: NewPrimitiveType("string", true),
		},
		"has?": {
			Name:      "has?",
			Namespace: "Env",
			Parameters: []FunctionParam{
				{Name: "key", Type: NewPrimitiveType("string", false)},
			},
			ReturnType: NewPrimitiveType("bool", false),
		},
	},
}

// LookupStdlibFunction looks up a standard library function by namespace and name
func LookupStdlibFunction(namespace, name string) (*Function, bool) {
	if namespace == "" {
		return nil, false
	}
	namespaceFuncs, ok := StdlibFunctions[namespace]
	if !ok {
		return nil, false
	}
	fn, ok := namespaceFuncs[name]
	return fn, ok
}
