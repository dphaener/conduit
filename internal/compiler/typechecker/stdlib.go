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

// StdlibFunctions contains MVP standard library function signatures (15 functions total)
// This is intentionally limited to the MVP scope - additional functions will be added later
var StdlibFunctions = map[string]map[string]*Function{
	"String": {
		"length": {
			Name:      "length",
			Namespace: "String",
			Parameters: []FunctionParam{
				{Name: "s", Type: NewPrimitiveType("string", false)},
			},
			ReturnType: NewPrimitiveType("int", false),
		},
		"slugify": {
			Name:      "slugify",
			Namespace: "String",
			Parameters: []FunctionParam{
				{Name: "s", Type: NewPrimitiveType("string", false)},
			},
			ReturnType: NewPrimitiveType("string", false),
		},
		"upcase": {
			Name:      "upcase",
			Namespace: "String",
			Parameters: []FunctionParam{
				{Name: "s", Type: NewPrimitiveType("string", false)},
			},
			ReturnType: NewPrimitiveType("string", false),
		},
		"downcase": {
			Name:      "downcase",
			Namespace: "String",
			Parameters: []FunctionParam{
				{Name: "s", Type: NewPrimitiveType("string", false)},
			},
			ReturnType: NewPrimitiveType("string", false),
		},
		"trim": {
			Name:      "trim",
			Namespace: "String",
			Parameters: []FunctionParam{
				{Name: "s", Type: NewPrimitiveType("string", false)},
			},
			ReturnType: NewPrimitiveType("string", false),
		},
		"contains": {
			Name:      "contains",
			Namespace: "String",
			Parameters: []FunctionParam{
				{Name: "s", Type: NewPrimitiveType("string", false)},
				{Name: "substr", Type: NewPrimitiveType("string", false)},
			},
			ReturnType: NewPrimitiveType("bool", false),
		},
		"replace": {
			Name:      "replace",
			Namespace: "String",
			Parameters: []FunctionParam{
				{Name: "s", Type: NewPrimitiveType("string", false)},
				{Name: "old", Type: NewPrimitiveType("string", false)},
				{Name: "new", Type: NewPrimitiveType("string", false)},
			},
			ReturnType: NewPrimitiveType("string", false),
		},
	},
	"Time": {
		"now": {
			Name:       "now",
			Namespace:  "Time",
			Parameters: []FunctionParam{},
			ReturnType: NewPrimitiveType("timestamp", false),
		},
		"format": {
			Name:      "format",
			Namespace: "Time",
			Parameters: []FunctionParam{
				{Name: "t", Type: NewPrimitiveType("timestamp", false)},
				{Name: "layout", Type: NewPrimitiveType("string", false)},
			},
			ReturnType: NewPrimitiveType("string", false),
		},
		"parse": {
			Name:      "parse",
			Namespace: "Time",
			Parameters: []FunctionParam{
				{Name: "s", Type: NewPrimitiveType("string", false)},
				{Name: "layout", Type: NewPrimitiveType("string", false)},
			},
			ReturnType: NewPrimitiveType("timestamp", true), // Returns nullable timestamp?
		},
		"add_days": {
			Name:      "add_days",
			Namespace: "Time",
			Parameters: []FunctionParam{
				{Name: "t", Type: NewPrimitiveType("timestamp", false)},
				{Name: "days", Type: NewPrimitiveType("int", false)},
			},
			ReturnType: NewPrimitiveType("timestamp", false),
		},
	},
	"Array": {
		"length": {
			Name:      "length",
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
				{Name: "value", Type: NewPrimitiveType("any", false)},
			},
			ReturnType: NewPrimitiveType("bool", false),
		},
	},
	"Hash": {
		"has_key": {
			Name:      "has_key",
			Namespace: "Hash",
			Parameters: []FunctionParam{
				{Name: "h", Type: NewHashType(NewPrimitiveType("any", false), NewPrimitiveType("any", false), false)},
				{Name: "key", Type: NewPrimitiveType("any", false)},
			},
			ReturnType: NewPrimitiveType("bool", false),
		},
	},
	"UUID": {
		"generate": {
			Name:       "generate",
			Namespace:  "UUID",
			Parameters: []FunctionParam{},
			ReturnType: NewPrimitiveType("uuid", false),
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
