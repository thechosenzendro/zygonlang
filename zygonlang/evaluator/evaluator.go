package evaluator

import (
	"bytes"
	"fmt"
	"os"
	"reflect"
	"strconv"
	"strings"
	"thechosenzendro/zygonlang/zygonlang/ast"
	"thechosenzendro/zygonlang/zygonlang/builtin"
	token "thechosenzendro/zygonlang/zygonlang/tokenizer"

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

var builtinLib = builtin.BuiltinLib()

func Eval(node ast.Node, env *Environment) Value {
	switch node := node.(type) {
	case ast.Program:
		var res Value
		for _, nd := range node.Body {
			run := true
			switch nd := nd.(type) {
			case ast.AssignmentStatement:
			case ast.FunctionDeclaration:
			case ast.UsingStatement:
			case ast.PubStatement:
			default:
				run = false
				res = Eval(nd, env)
				env.Set("_", res)
			}
			if run {
				res = Eval(nd, env)
			}

		}
		return res
	case ast.NumberLiteral:
		return Number(node)
	case ast.BooleanLiteral:
		return Boolean(node)
	case ast.TextLiteral:
		str := ""
		for _, part := range node.Parts {
			switch part := part.(type) {
			case ast.TextPart:
				str = str + part.Value
			default:
				str = str + Eval(part, env).Inspect()
			}
		}
		return Text{str}
	case ast.PrefixExpression:
		right := Eval(node.Right, env)
		switch node.Operator {
		case token.NOT:
			switch right := right.(type) {
			case Boolean:
				return Boolean{!right.Value}
			default:
				panic("non boolean passed to not")
			}
		case token.MINUS:
			switch right := right.(type) {
			case Number:
				return Number{-right.Value}
			}
		}
	case ast.InfixExpression:
		switch node.Operator {
		case token.IS:
			return Boolean{reflect.DeepEqual(Eval(node.Left, env), Eval(node.Right, env))}
		case token.IS_NOT:
			return Boolean{!reflect.DeepEqual(Eval(node.Left, env), Eval(node.Right, env))}
		case token.AND:
			return Boolean{Eval(node.Left, env).(Boolean).Value && Eval(node.Right, env).(Boolean).Value}
		case token.OR:
			left := Eval(node.Left, env)
			if left.Type() != BOOL {
				panic("left arg in or does not eval to a boolean")
			}
			if left.Inspect() == "true" {
				return Boolean{true}
			}
			right := Eval(node.Right, env)
			if right.Type() != BOOL {
				panic("right arg in or does not eval to a boolean")
			}
			if right.Inspect() == "true" {
				return Boolean{true}
			} else {
				return Boolean{false}
			}
		}
		left := Eval(node.Left, env)
		right := Eval(node.Right, env)

		if left.Type() == NUMBER && right.Type() == NUMBER {
			switch node.Operator {
			case token.PLUS:
				return Number{left.(Number).Value + right.(Number).Value}
			case token.MINUS:
				return Number{left.(Number).Value - right.(Number).Value}
			case token.STAR:
				return Number{left.(Number).Value * right.(Number).Value}
			case token.SLASH:
				return Number{left.(Number).Value / right.(Number).Value}
			case token.GREATER_THAN:
				return Boolean{left.(Number).Value > right.(Number).Value}
			case token.LESSER_THAN:
				return Boolean{left.(Number).Value < right.(Number).Value}

			}
		}

	case ast.Block:
		var res Value
		for _, nd := range node.Body {
			run := true
			switch nd := nd.(type) {
			case ast.AssignmentStatement:
			case ast.FunctionDeclaration:
			case ast.UsingStatement:
				panic("a using statement can only be at the top level")
			case ast.PubStatement:
				panic("a pub statement can only be at the top level")
			default:
				run = false
				res = Eval(nd, env)
				env.Set("_", res)
			}
			if run {
				res = Eval(nd, env)
			}
		}
		return res

	case ast.CaseExpression:
		var subject Value
		if node.Subject != nil {
			subject = Eval(node.Subject, env)
		}
	caseLoop:
		for _, _case := range node.Cases {
			var patternResult Value
			var patternEnviron Environment = Environment{Store: map[string]Value{}, Outer: env}

			if subject == nil {
				patternResult = Eval(_case.Pattern, env)

			} else {
				switch _pattern := _case.Pattern.(type) {
				case ast.TableLiteral:
					if subject.Type() == TABLE {
						ind := 0
						patternEnviron = Environment{Store: map[string]Value{}, Outer: env}
						usedKeys := map[Value]string{}
						for _, entry := range _pattern.Entries {
							var key Value
							if entry.Key == nil {
								key = Number{float64(ind)}
								ind += 1
							} else {
								key = TableKey(*entry.Key)
							}
							val, ok := subject.(Table).Entries.Get(key)
							if ok {
								usedKeys[key] = ""
								switch value := entry.Value.(type) {
								case ast.Identifier:
									patternEnviron.Set(value.Value, val)
									patternResult = Boolean{true}

								case ast.RestOperator:
									if value.Value == nil && subject.(Table).Entries.Len() < len(node.Cases) {
										if patternResult.(Boolean).Value {
											break caseLoop
										}
									} else {
										delete(usedKeys, key)
										table := Table{Entries: orderedmap.NewOrderedMap[Value, Value]()}
										i := 0
										for _, key := range subject.(Table).Entries.Keys() {
											if _, ok := usedKeys[key]; !ok {
												if key.Type() == NUMBER {
													key = Number{float64(i)}
													i += 1
												}
												value, _ := subject.(Table).Entries.Get(key)
												table.Entries.Set(key, value)
											}
										}
										patternEnviron.Set(value.Value.(ast.Identifier).Value, table)
									}
								default:
									if !reflect.DeepEqual(val, Eval(entry.Value, env)) {
										patternResult = Boolean{false}
										break caseLoop
									} else {
										patternResult = Boolean{true}
									}
								}
							} else {
								patternResult = Boolean{false}
								break caseLoop
							}
						}
					}
				default:
					patternEnviron = Environment{Store: map[string]Value{}, Outer: env}
					pattern := Eval(_pattern, env)
					patternResult = Boolean{reflect.DeepEqual(subject, pattern)}
				}
			}
			if patternResult == nil {
				panic("pattern does not eval to anything")
			}
			if patternResult.Type() != BOOL {
				panic("pattern result is not a boolean")
			}
			if patternResult.Inspect() == "true" {
				return Eval(_case.Block, &patternEnviron)
			}
		}
		if node.Default != nil {
			return Eval(*node.Default, env)
		}
		panic("No truthy case in case expr")
	case ast.AssignmentStatement:
		if _, ok := env.Get(node.Name.Value); !ok {
			val := Eval(node.Value, env)
			if val == nil {
				panic("value does not produce anything")
			}
			env.Set(node.Name.Value, val)
			return nil
		} else {
			panic(fmt.Sprintf("Cannot reassign identifier %s", node.Name.Value))
		}
	case ast.AccessOperator:
		subject := Eval(node.Subject, env)
		var index Value
		switch attribute := node.Attribute.(type) {
		case ast.Identifier:
			index = TableKey(attribute)
		case ast.Grouped:
			index = Eval(attribute.Value, env)
		}
		switch subject := subject.(type) {
		case Table:
			val, ok := subject.Entries.Get(index)
			if !ok {
				panic(fmt.Sprintf("bad index %s on table %s", index.Inspect(), subject.Inspect()))
			}
			return val

		default:
			panic(fmt.Sprintf("Cannot index type %T", subject))
		}
	case ast.Identifier:
		val, ok := env.Get(node.Value)
		if !ok {
			panic(fmt.Sprintf("identifier %s not found", node.Value))
		}
		return val
	case ast.FunctionDeclaration:
		fn := Function{Parameters: orderedmap.NewOrderedMap[TableKey, Value](), Body: node.Body, Rest: node.Rest, Env: env}

		for _, name := range node.Parameters.Keys() {
			param_default, _ := node.Parameters.Get(name)
			if param_default == nil {
				fn.Parameters.Set(TableKey(name), nil)
			} else {
				fn.Parameters.Set(TableKey(name), Eval(param_default, env))
			}

		}
		if node.Name != nil {
			env.Set(node.Name.Value, fn)
		}
		return fn
	case ast.FunctionCall:
		fn := Eval(node.Fn, env)
		switch function := fn.(type) {
		case Function:
			funcEnviron := &Environment{Store: make(map[string]Value), Outer: function.Env}
			i := 0
			for _, name := range function.Parameters.Keys() {
				param_default, _ := function.Parameters.Get(name)
				if len(node.Arguments) > i {
					arg := node.Arguments[i]
					switch value := arg.Value.(type) {
					case ast.RestOperator:
						_rest := Eval(value.Value, env)
						if _rest.Type() != TABLE {
							panic(fmt.Sprintf("cannot spread %T", _rest))
						}
						rest := _rest.(Table)
						for _, key := range rest.Entries.Keys() {
							value, _ := rest.Entries.Get(key)
							if key != nil {
								if _, ok := function.Parameters.Get(key.(TableKey)); ok {
									funcEnviron.Set(key.(TableKey).Value, value)
								} else {
									panic(fmt.Sprintf("Cannot spread items with names that arent in the parameters (%s)", key.Inspect()))
								}
							} else {
								panic("Cannot spread items without names")
							}
						}
					default:
						if arg.Name == nil {
							arg.Name = &name
						}
						var val Value = Eval(arg.Value, env)
						if arg.Value == nil {
							val = param_default
						}
						funcEnviron.Set(arg.Name.Value, val)

					}
				} else {
					name := &name
					value := param_default
					if value == nil {
						panic(fmt.Sprintf("no default for %s", name))
					}
					funcEnviron.Set(name.Value, value)
				}
				i += 1
			}
			if function.Rest != nil && function.Parameters.Len() < len(node.Arguments) {
				rest := Table{Entries: orderedmap.NewOrderedMap[Value, Value]()}
				ind := 0
				for _, arg := range node.Arguments[i:] {
					switch value := arg.Value.(type) {
					case ast.RestOperator:
						_rest := Eval(value.Value, env)
						if _rest.Type() != TABLE {
							panic(fmt.Sprintf("cannot spread %T", _rest))
						}
						rest := _rest.(Table)
						for _, key := range rest.Entries.Keys() {
							value, _ := rest.Entries.Get(key)
							if key != nil {
								if _, ok := function.Parameters.Get(key.(TableKey)); ok {
									funcEnviron.Set(key.(TableKey).Value, value)
								} else {
									panic("Cannot spread items with names that arent in the parameters")
								}
							} else {
								panic("Cannot spread items without names")
							}
						}
					default:
						if arg.Name != nil {
							rest.Entries.Set(TableKey(*arg.Name), Eval(arg.Value, env))
						} else {
							rest.Entries.Set(Number{float64(ind)}, Eval(arg.Value, env))
							ind += 1
						}
					}
				}
				funcEnviron.Set(function.Rest.Value.(ast.Identifier).Value, rest)
			}
			return Eval(function.Body, funcEnviron)

		case BuiltinFunction:
			i := 0
			funcEnviron := map[string]Value{}
			for _, name := range function.Contract.Parameters.Keys() {
				param_default, _ := function.Contract.Parameters.Get(name)
				if len(node.Arguments) > i {
					arg := node.Arguments[i]
					switch value := arg.Value.(type) {
					case ast.RestOperator:
						_rest := Eval(value.Value, env)
						if _rest.Type() != TABLE {
							panic(fmt.Sprintf("cannot spread %T", _rest))
						}
						rest := _rest.(Table)
						for _, key := range rest.Entries.Keys() {
							value, _ := rest.Entries.Get(key)
							if key != nil {
								if _, ok := function.Contract.Parameters.Get(key.(TableKey)); ok {
									funcEnviron[key.(TableKey).Value] = value
								} else {
									panic(fmt.Sprintf("Cannot spread items with names that arent in the parameters (%s)", key.Inspect()))
								}
							} else {
								panic("Cannot spread items without names")
							}
						}
					default:
						if arg.Name == nil {
							arg.Name = &name
						}
						var val Value = Eval(arg.Value, env)
						if arg.Value == nil {
							val = param_default
						}
						funcEnviron[arg.Name.Value] = val

					}
				} else {
					name := &name
					value := param_default
					if value == nil {
						panic(fmt.Sprintf("no default for %s", name))
					}
					funcEnviron[name.Value] = value
				}
				i += 1
			}
			if function.Contract.Rest != nil && function.Contract.Parameters.Len() < len(node.Arguments) {
				rest := Table{Entries: orderedmap.NewOrderedMap[Value, Value]()}
				ind := 0
				for _, arg := range node.Arguments[i:] {
					switch value := arg.Value.(type) {
					case ast.RestOperator:
						_rest := Eval(value.Value, env)
						if _rest.Type() != TABLE {
							panic(fmt.Sprintf("cannot spread %T", _rest))
						}
						rest := _rest.(Table)
						for _, key := range rest.Entries.Keys() {
							value, _ := rest.Entries.Get(key)
							if key != nil {
								if _, ok := function.Contract.Parameters.Get(key.(TableKey)); ok {
									funcEnviron[key.(TableKey).Value] = value
								} else {
									panic("Cannot spread items with names that arent in the parameters")
								}
							} else {
								panic("Cannot spread items without names")
							}
						}
					default:
						if arg.Name != nil {
							rest.Entries.Set(TableKey(*arg.Name), Eval(arg.Value, env))
						} else {
							rest.Entries.Set(Number{float64(ind)}, Eval(arg.Value, env))
							ind += 1
						}
					}
				}
				funcEnviron[function.Contract.Rest.Value.(ast.Identifier).Value] = rest
			}
			return function.Fn(funcEnviron)

		}
	case ast.TableLiteral:
		entries := orderedmap.NewOrderedMap[Value, Value]()
		index := -1
		for _, entry := range node.Entries {
			if entry.Key == nil {
				switch val := entry.Value.(type) {
				case ast.RestOperator:
					_table := Eval(val.Value, env)
					if _table.Type() != TABLE {
						panic("cannot spread non table values")
					}
					table := _table.(Table)
					for _, key := range table.Entries.Keys() {
						value, _ := table.Entries.Get(key)
						switch key := key.(type) {
						case TableKey:
							entries.Set(key, value)
						case Number:
							index += 1
							entries.Set(Number{float64(index)}, value)
						}
					}
				default:
					index += 1
					entries.Set(Number{float64(index)}, Eval(entry.Value, env))
				}
			} else {
				entries.Set(TableKey{entry.Key.Value}, Eval(entry.Value, env))
			}
		}
		return Table{entries}
	case ast.PubStatement:
		switch pub := node.Public.(type) {
		case ast.AssignmentStatement:
			Eval(pub, env)
			env.Set("pub "+pub.Name.Value, env.Store[pub.Name.Value])
		case ast.FunctionDeclaration:
			if pub.Name != nil {
				Eval(pub, env)
				env.Set("pub "+pub.Name.Value, env.Store[pub.Name.Value])
			} else {
				panic("anonymous function could not be made public")
			}
		default:
			panic(fmt.Sprintf("%T cannot be made public", pub))
		}
	case ast.UsingStatement:
		libRoot := "./lib"

		for _, module := range node.Modules {

			if builtin, ok := builtinLib.Get(module.Module); ok {
				unwrap(module.Module, Table{builtin}, env)
				for _, symbol := range module.Symbols {
					v, _ := builtin.Get(TableKey(symbol))
					env.Set(symbol.Value, v)
				}

			} else {
				var e *Environment
				var err error
				projectRoot := "./examples"
				_, e, err = getModule(projectRoot + getModPath(module.Module))
				if err != nil {
					_, e, err = getModule(libRoot + getModPath(module.Module))
					if err != nil {
						panic(err)
					}
				}
				pubTable := publicToTable(e)
				unwrap(module.Module, pubTable, env)
				for _, symbol := range module.Symbols {
					if val, ok := e.Get("pub " + symbol.Value); ok {
						env.Set(symbol.Value, val)
					}
				}

			}
		}
	case ast.RestOperator:
		panic("rest operator is not a normal expression and cant be used on its own. ")
	default:
		panic(fmt.Sprintf("eval error %T", node))
	}
	return nil
}

func getModule(modulePath string) (Value, *Environment, error) {
	source, err := os.ReadFile(modulePath)
	if err != nil {
		return nil, nil, fmt.Errorf("no module at %s", modulePath)
	}
	m, e := Exec(string(source))
	return m, e, nil

}

func publicToTable(e *Environment) Table {
	table := Table{Entries: orderedmap.NewOrderedMap[Value, Value]()}
	for key, value := range e.Store {
		if strings.HasPrefix(key, "pub ") {
			table.Entries.Set(TableKey{strings.SplitAfter(key, "pub ")[1]}, value)
		}
	}
	return table
}

func unwrap(m ast.Name, toPut Table, env *Environment) {
	switch m := m.(type) {
	case ast.AccessOperator:
		unwrap(m.Attribute.(ast.Name), toPut, env)
	case ast.Identifier:
		env.Set(m.Value, toPut)
	}
}

func getModPath(module ast.Name) string {
	switch mod := module.(type) {
	case ast.Identifier:
		return "/" + mod.Value + ".zygon"
	case ast.AccessOperator:
		return "/" + mod.Subject.(ast.Identifier).Value + "/" + getModPath(mod.Attribute.(ast.Name))
	}
	return ""
}

func Exec(sourceCode string) (Value, *Environment) {
	// the lexer needs to lex indents correctly
	tokens := token.Tokenize(sourceCode + "\n")
	ast := ast.Parse(&tokens)
	env := &Environment{Store: make(map[string]Value), Outer: nil}

	return Eval(ast, env), env
}
