package value

import (
	"bytes"
	"fmt"
	"strconv"
	"thechosenzendro/zygonlang/zygonlang/ast"

	"github.com/elliotchance/orderedmap/v2"
)

const (
	NUMBER    = "Number"
	BOOL      = "Boolean"
	TEXT      = "Text"
	FUNCTION  = "Function"
	TABLE     = "Table"
	TABLE_KEY = "TableKey"
	BUILTIN   = "BuiltinFunction"
	ERROR     = "Error"
)

type Value interface {
	Type() string
	Inspect() string
}

type Number struct {
	Value float64
}

func (n Number) Type() string    { return NUMBER }
func (n Number) Inspect() string { return strconv.FormatFloat(n.Value, 'f', -1, 64) }

type Boolean struct {
	Value bool
}

func (b Boolean) Type() string    { return BOOL }
func (b Boolean) Inspect() string { return fmt.Sprintf("%t", b.Value) }

type Text struct {
	Value string
}

func (t Text) Type() string    { return TEXT }
func (t Text) Inspect() string { return t.Value }

type Function struct {
	Parameters *orderedmap.OrderedMap[TableKey, Value]
	Body       ast.Block
	Rest       *ast.RestOperator
	Env        *Environment
}

func (f Function) Type() string    { return FUNCTION }
func (f Function) Inspect() string { return "Function Declaration" }

type Table struct {
	Entries *orderedmap.OrderedMap[Value, Value]
}

var tableIndentLevel int

func indent() string {
	str := ""
	for range tableIndentLevel {
		str = str + " "
	}
	return str
}

func (t Table) Type() string { return TABLE }
func (t Table) Inspect() string {
	var out bytes.Buffer
	out.WriteString("{\n")
	tableIndentLevel += 4
	for _, key := range t.Entries.Keys() {
		value, _ := t.Entries.Get(key)
		var k string
		var v string
		switch key.(type) {
		case Text:
			k = "\"" + key.Inspect() + "\""
		default:
			k = key.Inspect()
		}

		switch value.(type) {
		case Text:
			v = "\"" + value.Inspect() + "\""
		default:
			v = value.Inspect()
		}

		out.WriteString(fmt.Sprintf("%s%s: %s\n", indent(), k, v))
	}
	tableIndentLevel -= 4
	out.WriteString(fmt.Sprintf("%s}\n", indent()))
	return out.String()
}

type TableKey struct {
	Value string
}

func (tk TableKey) Type() string    { return TABLE_KEY }
func (tk TableKey) Inspect() string { return tk.Value }

type BuiltinFunctionContract struct {
	Parameters *orderedmap.OrderedMap[TableKey, Value]
	Rest       *ast.RestOperator
}

type BuiltinFunction struct {
	Contract BuiltinFunctionContract
	Fn       func(args map[string]Value) Value
}

func (b BuiltinFunction) Type() string    { return BUILTIN }
func (b BuiltinFunction) Inspect() string { return "Builtin" }

type Error struct {
	Value string
}

func (e Error) Type() string    { return ERROR }
func (e Error) Inspect() string { return fmt.Sprintf("Error(%s)", e.Value) }

type Environment struct {
	Store map[string]Value
	Outer *Environment
}

func (e *Environment) Get(name string) (Value, bool) {
	val, ok := e.Store[name]
	if !ok && e.Outer != nil {
		val, ok = e.Outer.Get(name)
	}
	return val, ok
}

func (e *Environment) Set(name string, val Value) Value {
	e.Store[name] = val
	return val
}
