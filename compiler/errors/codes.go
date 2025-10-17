package errors

// Error code constants organized by phase
// E001-E099: Lexer errors
// E100-E199: Parser errors
// E200-E299: Type errors
// E300-E399: Constraint errors
// E400-E499: Codegen errors

const (
	// Lexer errors (E001-E099)
	ErrUnterminatedString     = "E001"
	ErrInvalidCharacter       = "E002"
	ErrInvalidNumber          = "E003"
	ErrUnterminatedComment    = "E004"
	ErrInvalidEscape          = "E005"
	ErrInvalidUnicode         = "E006"
	ErrNumberOverflow         = "E007"
	ErrInvalidHexNumber       = "E008"
	ErrInvalidBinaryNumber    = "E009"
	ErrInvalidOctalNumber     = "E010"
	ErrUnexpectedEOF          = "E011"
	ErrInvalidIdentifier      = "E012"
	ErrInvalidOperator        = "E013"
	ErrInvalidAnnotation      = "E014"
	ErrMalformedToken         = "E015"

	// Parser errors (E100-E199)
	ErrUnexpectedToken        = "E100"
	ErrExpectedIdentifier     = "E101"
	ErrExpectedType           = "E102"
	ErrExpectedColon          = "E103"
	ErrExpectedBrace          = "E104"
	ErrExpectedParen          = "E105"
	ErrExpectedBracket        = "E106"
	ErrMissingNullability     = "E107"
	ErrInvalidFieldDefinition = "E108"
	ErrInvalidResourceDef     = "E109"
	ErrInvalidRelationship    = "E110"
	ErrInvalidConstraint      = "E111"
	ErrInvalidHook            = "E112"
	ErrInvalidExpression      = "E113"
	ErrInvalidStatement       = "E114"
	ErrDuplicateField         = "E115"
	ErrDuplicateResource      = "E116"
	ErrInvalidParserAnnotation = "E117"
	ErrMissingBlock           = "E118"
	ErrInvalidBlockContent    = "E119"
	ErrInvalidEnumValue       = "E120"
	ErrInvalidArrayType       = "E121"
	ErrInvalidHashType        = "E122"
	ErrEmptyEnum              = "E123"
	ErrInvalidParserMetadata  = "E124"
	ErrMissingRequired        = "E125"
	ErrInvalidSyntax          = "E126"
	ErrUnmatchedBrace         = "E127"
	ErrUnmatchedParen         = "E128"
	ErrUnmatchedBracket       = "E129"
	ErrInvalidAssignment      = "E130"

	// Type errors (E200-E299)
	ErrTypeMismatch           = "E200"
	ErrUndefinedType          = "E201"
	ErrUndefinedResource      = "E202"
	ErrUndefinedField         = "E203"
	ErrUndefinedFunction      = "E204"
	ErrInvalidOperation       = "E205"
	ErrIncompatibleTypes      = "E206"
	ErrInvalidCast            = "E207"
	ErrNullabilityViolation   = "E208"
	ErrCircularDependency     = "E209"
	ErrInvalidFunctionCall    = "E210"
	ErrWrongArgumentCount     = "E211"
	ErrInvalidArgumentType    = "E212"
	ErrInvalidReturnType      = "E213"
	ErrInvalidComparison      = "E214"
	ErrInvalidArithmetic      = "E215"
	ErrInvalidLogical         = "E216"
	ErrInvalidIndexing        = "E217"
	ErrInvalidFieldAccess     = "E218"
	ErrReadOnlyAssignment     = "E219"
	ErrInvalidEnumAccess      = "E220"
	ErrAmbiguousType          = "E221"
	ErrInvalidNamespace       = "E222"
	ErrPrivateAccess          = "E223"
	ErrInvalidSelfReference   = "E224"
	ErrInvalidContextAccess   = "E225"
	ErrInvalidArrayElement    = "E226"
	ErrInvalidHashKey         = "E227"
	ErrInvalidHashValue       = "E228"
	ErrTypeNotComparable      = "E229"
	ErrTypeNotIterable        = "E230"

	// Constraint errors (E300-E399)
	ErrConstraintViolation    = "E300"
	ErrMinValueViolation      = "E301"
	ErrMaxValueViolation      = "E302"
	ErrUniqueViolation        = "E303"
	ErrPatternViolation       = "E304"
	ErrRequiredViolation      = "E305"
	ErrDefaultTypeMismatch    = "E306"
	ErrInvalidConstraintArg   = "E307"
	ErrConflictingConstraints = "E308"
	ErrInvalidMinMax          = "E309"
	ErrInvalidPattern         = "E310"
	ErrInvalidDefault         = "E311"
	ErrPrimaryKeyConflict     = "E312"
	ErrAutoIncrementInvalid   = "E313"
	ErrForeignKeyInvalid      = "E314"
	ErrOnDeleteInvalid        = "E315"
	ErrOnUpdateInvalid        = "E316"
	ErrInvariantViolation     = "E317"
	ErrInvalidCondition       = "E318"
	ErrCustomConstraintFailed = "E319"
	ErrConstraintNotFound     = "E320"

	// Codegen errors (E400-E499)
	ErrCodegenFailed          = "E400"
	ErrInvalidGoCode          = "E401"
	ErrDuplicateGeneration    = "E402"
	ErrMissingTemplate        = "E403"
	ErrTemplateError          = "E404"
	ErrInvalidImport          = "E405"
	ErrCyclicImport           = "E406"
	ErrGenerationConflict     = "E407"
	ErrInvalidPackageName     = "E408"
	ErrInvalidStructName      = "E409"
	ErrInvalidMethodName      = "E410"
	ErrInvalidFieldName       = "E411"
	ErrGoKeywordConflict      = "E412"
	ErrInvalidDBTag           = "E413"
	ErrInvalidJSONTag         = "E414"
	ErrMissingMetadata        = "E415"
	ErrInvalidMetadata        = "E416"
	ErrInvalidValidation      = "E417"
	ErrInvalidHookGeneration  = "E418"
	ErrInvalidQueryGeneration = "E419"
	ErrInvalidHandlerGen      = "E420"
)

// ErrorMessages maps error codes to their default messages
var ErrorMessages = map[string]string{
	// Lexer errors
	ErrUnterminatedString:     "Unterminated string literal",
	ErrInvalidCharacter:       "Invalid character",
	ErrInvalidNumber:          "Invalid number format",
	ErrUnterminatedComment:    "Unterminated comment",
	ErrInvalidEscape:          "Invalid escape sequence",
	ErrInvalidUnicode:         "Invalid unicode escape sequence",
	ErrNumberOverflow:         "Number overflow",
	ErrInvalidHexNumber:       "Invalid hexadecimal number",
	ErrInvalidBinaryNumber:    "Invalid binary number",
	ErrInvalidOctalNumber:     "Invalid octal number",
	ErrUnexpectedEOF:          "Unexpected end of file",
	ErrInvalidIdentifier:      "Invalid identifier",
	ErrInvalidOperator:        "Invalid operator",
	ErrInvalidAnnotation:      "Invalid annotation",
	ErrMalformedToken:         "Malformed token",

	// Parser errors
	ErrUnexpectedToken:        "Unexpected token",
	ErrExpectedIdentifier:     "Expected identifier",
	ErrExpectedType:           "Expected type",
	ErrExpectedColon:          "Expected ':'",
	ErrExpectedBrace:          "Expected '{' or '}'",
	ErrExpectedParen:          "Expected '(' or ')'",
	ErrExpectedBracket:        "Expected '[' or ']'",
	ErrMissingNullability:     "Missing nullability marker (! or ?)",
	ErrInvalidFieldDefinition: "Invalid field definition",
	ErrInvalidResourceDef:     "Invalid resource definition",
	ErrInvalidRelationship:    "Invalid relationship definition",
	ErrInvalidConstraint:      "Invalid constraint",
	ErrInvalidHook:            "Invalid hook definition",
	ErrInvalidExpression:      "Invalid expression",
	ErrInvalidStatement:       "Invalid statement",
	ErrDuplicateField:         "Duplicate field name",
	ErrDuplicateResource:      "Duplicate resource name",
	ErrInvalidParserAnnotation: "Invalid annotation",
	ErrMissingBlock:           "Missing block",
	ErrInvalidBlockContent:    "Invalid block content",
	ErrInvalidEnumValue:       "Invalid enum value",
	ErrInvalidArrayType:       "Invalid array type definition",
	ErrInvalidHashType:        "Invalid hash type definition",
	ErrEmptyEnum:              "Enum must have at least one value",
	ErrInvalidParserMetadata:  "Invalid metadata",
	ErrMissingRequired:        "Missing required element",
	ErrInvalidSyntax:          "Invalid syntax",
	ErrUnmatchedBrace:         "Unmatched brace",
	ErrUnmatchedParen:         "Unmatched parenthesis",
	ErrUnmatchedBracket:       "Unmatched bracket",
	ErrInvalidAssignment:      "Invalid assignment",

	// Type errors
	ErrTypeMismatch:           "Type mismatch",
	ErrUndefinedType:          "Undefined type",
	ErrUndefinedResource:      "Undefined resource",
	ErrUndefinedField:         "Undefined field",
	ErrUndefinedFunction:      "Undefined function",
	ErrInvalidOperation:       "Invalid operation",
	ErrIncompatibleTypes:      "Incompatible types",
	ErrInvalidCast:            "Invalid type cast",
	ErrNullabilityViolation:   "Nullability violation",
	ErrCircularDependency:     "Circular dependency detected",
	ErrInvalidFunctionCall:    "Invalid function call",
	ErrWrongArgumentCount:     "Wrong number of arguments",
	ErrInvalidArgumentType:    "Invalid argument type",
	ErrInvalidReturnType:      "Invalid return type",
	ErrInvalidComparison:      "Invalid comparison",
	ErrInvalidArithmetic:      "Invalid arithmetic operation",
	ErrInvalidLogical:         "Invalid logical operation",
	ErrInvalidIndexing:        "Invalid indexing operation",
	ErrInvalidFieldAccess:     "Invalid field access",
	ErrReadOnlyAssignment:     "Cannot assign to read-only value",
	ErrInvalidEnumAccess:      "Invalid enum access",
	ErrAmbiguousType:          "Ambiguous type",
	ErrInvalidNamespace:       "Invalid namespace",
	ErrPrivateAccess:          "Cannot access private member",
	ErrInvalidSelfReference:   "Invalid use of 'self'",
	ErrInvalidContextAccess:   "Invalid context access",
	ErrInvalidArrayElement:    "Invalid array element type",
	ErrInvalidHashKey:         "Invalid hash key type",
	ErrInvalidHashValue:       "Invalid hash value type",
	ErrTypeNotComparable:      "Type is not comparable",
	ErrTypeNotIterable:        "Type is not iterable",

	// Constraint errors
	ErrConstraintViolation:    "Constraint violation",
	ErrMinValueViolation:      "Value below minimum",
	ErrMaxValueViolation:      "Value exceeds maximum",
	ErrUniqueViolation:        "Uniqueness constraint violated",
	ErrPatternViolation:       "Pattern constraint violated",
	ErrRequiredViolation:      "Required field missing",
	ErrDefaultTypeMismatch:    "Default value type mismatch",
	ErrInvalidConstraintArg:   "Invalid constraint argument",
	ErrConflictingConstraints: "Conflicting constraints",
	ErrInvalidMinMax:          "Invalid min/max constraint",
	ErrInvalidPattern:         "Invalid pattern",
	ErrInvalidDefault:         "Invalid default value",
	ErrPrimaryKeyConflict:     "Multiple primary keys defined",
	ErrAutoIncrementInvalid:   "Auto-increment only valid on integer types",
	ErrForeignKeyInvalid:      "Invalid foreign key",
	ErrOnDeleteInvalid:        "Invalid on_delete action",
	ErrOnUpdateInvalid:        "Invalid on_update action",
	ErrInvariantViolation:     "Invariant violated",
	ErrInvalidCondition:       "Invalid condition",
	ErrCustomConstraintFailed: "Custom constraint failed",
	ErrConstraintNotFound:     "Constraint not found",

	// Codegen errors
	ErrCodegenFailed:          "Code generation failed",
	ErrInvalidGoCode:          "Generated invalid Go code",
	ErrDuplicateGeneration:    "Duplicate code generation",
	ErrMissingTemplate:        "Missing template",
	ErrTemplateError:          "Template error",
	ErrInvalidImport:          "Invalid import",
	ErrCyclicImport:           "Cyclic import detected",
	ErrGenerationConflict:     "Code generation conflict",
	ErrInvalidPackageName:     "Invalid package name",
	ErrInvalidStructName:      "Invalid struct name",
	ErrInvalidMethodName:      "Invalid method name",
	ErrInvalidFieldName:       "Invalid field name",
	ErrGoKeywordConflict:      "Conflicts with Go keyword",
	ErrInvalidDBTag:           "Invalid database tag",
	ErrInvalidJSONTag:         "Invalid JSON tag",
	ErrMissingMetadata:        "Missing metadata",
	ErrInvalidMetadata:        "Invalid metadata",
	ErrInvalidValidation:      "Invalid validation code",
	ErrInvalidHookGeneration:  "Invalid hook generation",
	ErrInvalidQueryGeneration: "Invalid query generation",
	ErrInvalidHandlerGen:      "Invalid handler generation",
}

// GetErrorMessage returns the default message for an error code
func GetErrorMessage(code string) string {
	if msg, ok := ErrorMessages[code]; ok {
		return msg
	}
	return "Unknown error"
}

// GetPhaseForCode returns the phase name for an error code
func GetPhaseForCode(code string) string {
	if len(code) < 2 {
		return "unknown"
	}

	// Extract the numeric part
	if code[0] != 'E' {
		return "unknown"
	}

	// Determine phase based on error code range
	switch {
	case code >= "E001" && code <= "E099":
		return "lexer"
	case code >= "E100" && code <= "E199":
		return "parser"
	case code >= "E200" && code <= "E299":
		return "type_checker"
	case code >= "E300" && code <= "E399":
		return "constraint"
	case code >= "E400" && code <= "E499":
		return "codegen"
	default:
		return "unknown"
	}
}
