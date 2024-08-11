package types

import (
	"bytes"
	"fmt"

	"github.com/elliotchance/orderedmap/v2"
)

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

type Type struct {
	Base       BaseType
	Properties *orderedmap.OrderedMap[string, *Type]
}

func indent(tableIndentLevel int) string {
	str := ""
	for range tableIndentLevel {
		str = str + " "
	}
	return str
}

func (t *Type) Inspect(indentLevel int) string {
	var out bytes.Buffer
	out.WriteString(string(t.Base))
	if t.Properties != nil {
		out.WriteString(" {")
		out.WriteString("\n")
		for _, key := range t.Properties.Keys() {
			typ, _ := t.Properties.Get(key)
			var t string
			if typ != nil {
				t = typ.Inspect(indentLevel + 4)
			} else {
				t = "Any"
			}
			out.WriteString(fmt.Sprintf("%s%s: %s,\n", indent(indentLevel+4), key, t))
		}
		out.WriteString("}")
	}
	return out.String()
}

func NewType(base BaseType, properties *orderedmap.OrderedMap[string, *Type]) *Type {
	return &Type{Base: base, Properties: properties}
}

type TypeEnvironment struct {
	Store map[string]*Type
	Outer *TypeEnvironment
}

func (t *TypeEnvironment) Get(name string) (*Type, bool) {
	val, ok := t.Store[name]
	if !ok && t.Outer != nil {
		val, ok = t.Outer.Get(name)
	}
	return val, ok
}

func (t *TypeEnvironment) Set(name string, typ *Type) *Type {
	t.Store[name] = typ
	return typ
}
