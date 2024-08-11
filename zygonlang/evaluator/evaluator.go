package evaluator

import (
	"fmt"
	"os"
	"reflect"
	"strings"
	"thechosenzendro/zygonlang/zygonlang/analyzer"
	"thechosenzendro/zygonlang/zygonlang/ast"
	"thechosenzendro/zygonlang/zygonlang/builtin"
	"thechosenzendro/zygonlang/zygonlang/token"
	"thechosenzendro/zygonlang/zygonlang/types"
	"thechosenzendro/zygonlang/zygonlang/value"

	"github.com/elliotchance/orderedmap/v2"
)

var builtinLib = builtin.BuiltinLib()

func Eval(node ast.Node, env *value.Environment) value.Value {
	switch node := node.(type) {
	case ast.Program:
		var res value.Value
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
		return value.Number(node)
	case ast.BooleanLiteral:
		return value.Boolean(node)
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
		return value.Text{Value: str}
	case ast.PrefixExpression:
		right := Eval(node.Right, env)
		switch node.Operator {
		case token.NOT:
			switch right := right.(type) {
			case value.Boolean:
				return value.Boolean{Value: !right.Value}
			default:
				panic("non boolean passed to not")
			}
		case token.MINUS:
			switch right := right.(type) {
			case value.Number:
				return value.Number{Value: -right.Value}
			}
		}
	case ast.InfixExpression:
		switch node.Operator {
		case token.IS:
			return value.Boolean{Value: reflect.DeepEqual(Eval(node.Left, env), Eval(node.Right, env))}
		case token.IS_NOT:
			return value.Boolean{Value: !reflect.DeepEqual(Eval(node.Left, env), Eval(node.Right, env))}
		case token.AND:
			return value.Boolean{Value: Eval(node.Left, env).(value.Boolean).Value && Eval(node.Right, env).(value.Boolean).Value}
		case token.OR:
			left := Eval(node.Left, env)
			if left.Type() != types.BOOL {
				panic("left arg in or does not eval to a boolean")
			}
			if left.Inspect() == "true" {
				return value.Boolean{Value: true}
			}
			right := Eval(node.Right, env)
			if right.Type() != types.BOOL {
				panic("right arg in or does not eval to a boolean")
			}
			if right.Inspect() != "true" {
				return value.Boolean{Value: false}
			}

			return value.Boolean{Value: true}
		}
		left := Eval(node.Left, env)
		right := Eval(node.Right, env)

		if left.Type() == types.NUMBER && right.Type() == types.NUMBER {
			switch node.Operator {
			case token.PLUS:
				return value.Number{Value: left.(value.Number).Value + right.(value.Number).Value}
			case token.MINUS:
				return value.Number{Value: left.(value.Number).Value - right.(value.Number).Value}
			case token.STAR:
				return value.Number{Value: left.(value.Number).Value * right.(value.Number).Value}
			case token.SLASH:
				return value.Number{Value: left.(value.Number).Value / right.(value.Number).Value}
			case token.GREATER_THAN:
				return value.Boolean{Value: left.(value.Number).Value > right.(value.Number).Value}
			case token.LESSER_THAN:
				return value.Boolean{Value: left.(value.Number).Value < right.(value.Number).Value}

			}
		}

	case ast.Block:
		var res value.Value
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
		var subject value.Value
		if node.Subject != nil {
			subject = Eval(node.Subject, env)
		}
	caseLoop:
		for _, _case := range node.Cases {
			var patternResult value.Value
			var patternEnviron value.Environment = value.Environment{Store: map[string]value.Value{}, Outer: env}

			if subject == nil {
				patternResult = Eval(_case.Pattern, env)

			} else {
				switch _pattern := _case.Pattern.(type) {
				case ast.TableLiteral:
					if subject.Type() == types.TABLE {
						ind := 0
						patternEnviron = value.Environment{Store: map[string]value.Value{}, Outer: env}
						usedKeys := map[value.Value]string{}
						for _, entry := range _pattern.Entries {
							var key value.Value
							if entry.Key == nil {
								key = value.Number{Value: float64(ind)}
								ind += 1
							} else {
								key = value.TableKey(*entry.Key)
							}
							val, ok := subject.(value.Table).Entries.Get(key)
							if ok {
								usedKeys[key] = ""
								switch entryValue := entry.Value.(type) {
								case ast.Identifier:
									patternEnviron.Set(entryValue.Value, val)
									patternResult = value.Boolean{Value: true}

								case ast.RestOperator:
									if entryValue.Value == nil && subject.(value.Table).Entries.Len() < len(node.Cases) {
										if patternResult.(value.Boolean).Value {
											break caseLoop
										}
									} else {
										delete(usedKeys, key)
										table := value.Table{Entries: orderedmap.NewOrderedMap[value.Value, value.Value]()}
										i := 0
										for _, key := range subject.(value.Table).Entries.Keys() {
											if _, ok := usedKeys[key]; !ok {
												if key.Type() == types.NUMBER {
													key = value.Number{Value: float64(i)}
													i += 1
												}
												value, _ := subject.(value.Table).Entries.Get(key)
												table.Entries.Set(key, value)
											}
										}
										patternEnviron.Set(entryValue.Value.(ast.Identifier).Value, table)
									}
								default:
									if !reflect.DeepEqual(val, Eval(entry.Value, env)) {
										patternResult = value.Boolean{Value: false}
										break caseLoop
									} else {
										patternResult = value.Boolean{Value: true}
									}
								}
							} else {
								patternResult = value.Boolean{Value: false}
								break caseLoop
							}
						}
					}
				default:
					patternEnviron = value.Environment{Store: map[string]value.Value{}, Outer: env}
					pattern := Eval(_pattern, env)
					patternResult = value.Boolean{Value: reflect.DeepEqual(subject, pattern)}
				}
			}
			if patternResult == nil {
				panic("pattern does not eval to anything")
			}
			if patternResult.Type() != types.BOOL {
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
		var index value.Value
		switch attribute := node.Attribute.(type) {
		case ast.Identifier:
			index = value.TableKey(attribute)
		case ast.Grouped:
			index = Eval(attribute.Value, env)
		}
		switch subject := subject.(type) {
		case value.Table:
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
		fn := value.Function{Parameters: orderedmap.NewOrderedMap[value.TableKey, value.Value](), Body: node.Body, Rest: node.Rest, Env: env}

		for _, name := range node.Parameters.Keys() {
			param_default, _ := node.Parameters.Get(name)
			if param_default == nil {
				fn.Parameters.Set(value.TableKey(name), nil)
			} else {
				fn.Parameters.Set(value.TableKey(name), Eval(param_default, env))
			}

		}
		if node.Name != nil {
			env.Set(node.Name.Value, fn)
		}
		return fn
	case ast.FunctionCall:
		fn := Eval(node.Fn, env)
		switch function := fn.(type) {
		case value.Function:
			funcEnviron := &value.Environment{Store: make(map[string]value.Value), Outer: function.Env}
			i := 0
			for _, name := range function.Parameters.Keys() {
				param_default, _ := function.Parameters.Get(name)
				if len(node.Arguments) > i {
					arg := node.Arguments[i]
					switch argValue := arg.Value.(type) {
					case ast.RestOperator:
						_rest := Eval(argValue.Value, env)
						if _rest.Type() != types.TABLE {
							panic(fmt.Sprintf("cannot spread %T", _rest))
						}
						rest := _rest.(value.Table)
						for _, key := range rest.Entries.Keys() {
							entryValue, _ := rest.Entries.Get(key)
							if key != nil {
								if _, ok := function.Parameters.Get(key.(value.TableKey)); ok {
									funcEnviron.Set(key.(value.TableKey).Value, entryValue)
								} else {
									panic(fmt.Sprintf("Cannot spread items with names that arent in the parameters (%s)", key.Inspect()))
								}
							} else {
								panic("Cannot spread items without names")
							}
						}
					default:
						if arg.Name == nil {
							arg.Name = &ast.Identifier{Value: name.Value}
						}
						var val value.Value = Eval(arg.Value, env)
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
				rest := value.Table{Entries: orderedmap.NewOrderedMap[value.Value, value.Value]()}
				ind := 0
				for _, arg := range node.Arguments[i:] {
					switch argValue := arg.Value.(type) {
					case ast.RestOperator:
						_rest := Eval(argValue.Value, env)
						if _rest.Type() != types.TABLE {
							panic(fmt.Sprintf("cannot spread %T", _rest))
						}
						rest := _rest.(value.Table)
						for _, key := range rest.Entries.Keys() {
							entryValue, _ := rest.Entries.Get(key)
							if key != nil {
								if _, ok := function.Parameters.Get(key.(value.TableKey)); ok {
									funcEnviron.Set(key.(value.TableKey).Value, entryValue)
								} else {
									panic("Cannot spread items with names that arent in the parameters")
								}
							} else {
								panic("Cannot spread items without names")
							}
						}
					default:
						if arg.Name != nil {
							rest.Entries.Set(value.TableKey(*arg.Name), Eval(arg.Value, env))
						} else {
							rest.Entries.Set(value.Number{Value: float64(ind)}, Eval(arg.Value, env))
							ind += 1
						}
					}
				}
				funcEnviron.Set(function.Rest.Value.(ast.Identifier).Value, rest)
			}
			return Eval(function.Body, funcEnviron)

		case value.BuiltinFunction:
			i := 0
			funcEnviron := map[string]value.Value{}
			for _, name := range function.Contract.Parameters.Keys() {
				param_default, _ := function.Contract.Parameters.Get(name)
				if len(node.Arguments) > i {
					arg := node.Arguments[i]
					switch argValue := arg.Value.(type) {
					case ast.RestOperator:
						_rest := Eval(argValue.Value, env)
						if _rest.Type() != types.TABLE {
							panic(fmt.Sprintf("cannot spread %T", _rest))
						}
						rest := _rest.(value.Table)
						for _, key := range rest.Entries.Keys() {
							entryValue, _ := rest.Entries.Get(key)
							if key != nil {
								if _, ok := function.Contract.Parameters.Get(key.(value.TableKey)); ok {
									funcEnviron[key.(value.TableKey).Value] = entryValue
								} else {
									panic(fmt.Sprintf("Cannot spread items with names that arent in the parameters (%s)", key.Inspect()))
								}
							} else {
								panic("Cannot spread items without names")
							}
						}
					default:
						if arg.Name == nil {
							arg.Name = &ast.Identifier{Value: name.Value}
						}
						var val value.Value = Eval(arg.Value, env)
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
				rest := value.Table{Entries: orderedmap.NewOrderedMap[value.Value, value.Value]()}
				ind := 0
				for _, arg := range node.Arguments[i:] {
					switch argValue := arg.Value.(type) {
					case ast.RestOperator:
						_rest := Eval(argValue.Value, env)
						if _rest.Type() != types.TABLE {
							panic(fmt.Sprintf("cannot spread %T", _rest))
						}
						rest := _rest.(value.Table)
						for _, key := range rest.Entries.Keys() {
							entryValue, _ := rest.Entries.Get(key)
							if key != nil {
								if _, ok := function.Contract.Parameters.Get(key.(value.TableKey)); ok {
									funcEnviron[key.(value.TableKey).Value] = entryValue
								} else {
									panic("Cannot spread items with names that arent in the parameters")
								}
							} else {
								panic("Cannot spread items without names")
							}
						}
					default:
						if arg.Name != nil {
							rest.Entries.Set(value.TableKey(*arg.Name), Eval(arg.Value, env))
						} else {
							rest.Entries.Set(value.Number{Value: float64(ind)}, Eval(arg.Value, env))
							ind += 1
						}
					}
				}
				funcEnviron[function.Contract.Rest.Value.(ast.Identifier).Value] = rest
			}
			return function.Fn(funcEnviron)

		}
	case ast.TableLiteral:
		entries := orderedmap.NewOrderedMap[value.Value, value.Value]()
		index := -1
		for _, entry := range node.Entries {
			if entry.Key == nil {
				switch val := entry.Value.(type) {
				case ast.RestOperator:
					_table := Eval(val.Value, env)
					if _table.Type() != types.TABLE {
						panic("cannot spread non table values")
					}
					table := _table.(value.Table)
					for _, key := range table.Entries.Keys() {
						entryValue, _ := table.Entries.Get(key)
						switch key := key.(type) {
						case value.TableKey:
							entries.Set(key, entryValue)
						case value.Number:
							index += 1
							entries.Set(value.Number{Value: float64(index)}, entryValue)
						}
					}
				default:
					index += 1
					entries.Set(value.Number{Value: float64(index)}, Eval(entry.Value, env))
				}
			} else {
				entries.Set(value.TableKey{Value: entry.Key.Value}, Eval(entry.Value, env))
			}
		}
		return value.Table{Entries: entries}
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
				unwrap(module.Module, value.Table{Entries: builtin}, env)
				for _, symbol := range module.Symbols {
					v, _ := builtin.Get(value.TableKey(symbol))
					env.Set(symbol.Value, v)
				}

			} else {
				var e *value.Environment
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

func getModule(modulePath string) (value.Value, *value.Environment, error) {
	source, err := os.ReadFile(modulePath)
	if err != nil {
		return nil, nil, fmt.Errorf("no module at %s", modulePath)
	}
	m, e := Exec(string(source))
	return m, e, nil

}

func publicToTable(e *value.Environment) value.Table {
	table := value.Table{Entries: orderedmap.NewOrderedMap[value.Value, value.Value]()}
	for key, storeValue := range e.Store {
		if strings.HasPrefix(key, "pub ") {
			table.Entries.Set(value.TableKey{Value: strings.SplitAfter(key, "pub ")[1]}, storeValue)
		}
	}
	return table
}

func unwrap(m ast.Name, toPut value.Table, env *value.Environment) {
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

func Exec(sourceCode string) (value.Value, *value.Environment) {
	// the lexer needs to lex indents correctly
	tokens := token.Tokenize(sourceCode + "\n")

	ast := ast.Parse(&tokens)
	ast = analyzer.Analyze(ast)

	env := &value.Environment{Store: make(map[string]value.Value), Outer: nil}
	return Eval(ast, env), env
}
