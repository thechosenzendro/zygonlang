package types

const (
	NUMBER    = "Number"
	BOOL      = "Boolean"
	TEXT      = "Text"
	FUNCTION  = "Function"
	TABLE     = "Table"
	TABLE_KEY = "TableKey"
	TYPE      = "Type"
	BUILTIN   = "BuiltinFunction"
	ERROR     = "Error"
)

type BaseType string

type Variant struct {
	Base       BaseType
	Properties map[string]BaseType
}
