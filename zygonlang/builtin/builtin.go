package builtin

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"thechosenzendro/zygonlang/zygonlang/ast"
	ordmap "thechosenzendro/zygonlang/zygonlang/orderedmap"
	"thechosenzendro/zygonlang/zygonlang/types"
	"thechosenzendro/zygonlang/zygonlang/value"

	"github.com/elliotchance/orderedmap/v2"
	"github.com/tiendc/go-deepcopy"
)

func BuiltinLib() *orderedmap.OrderedMap[ast.Name, *orderedmap.OrderedMap[value.Value, value.Value]] {
	builtinLib := orderedmap.NewOrderedMap[ast.Name, *orderedmap.OrderedMap[value.Value, value.Value]]()
	// IO module
	ioModule := orderedmap.NewOrderedMap[value.Value, value.Value]()
	// IO.log
	ioModule.Set(
		value.TableKey{Value: "log"},
		value.BuiltinFunction{
			Contract: value.BuiltinFunctionContract{
				Parameters: ordmap.OrderedMapFromArgs([]ordmap.KV[value.TableKey, value.Value]{
					{Key: value.TableKey{Value: "message"}, Value: nil},
				}),
				Rest: nil,
			},

			Fn: func(args map[string]value.Value) value.Value {
				fmt.Print(args["message"].Inspect() + "\n")
				return nil
			},
		})
	// IO.get
	ioModule.Set(
		value.TableKey{Value: "get"},
		value.BuiltinFunction{
			Contract: value.BuiltinFunctionContract{
				Parameters: ordmap.OrderedMapFromArgs([]ordmap.KV[value.TableKey, value.Value]{
					{Key: value.TableKey{Value: "prompt"}, Value: nil},
				}),
				Rest: nil,
			},
			Fn: func(args map[string]value.Value) value.Value {
				prompt := args["prompt"].Inspect()
				fmt.Print(prompt)
				var input string
				scanner := bufio.NewScanner(os.Stdin)
				if scanner.Scan() {
					input = scanner.Text()
				}
				return value.Text{Value: input}
			},
		},
	)

	// Table module
	tableModule := orderedmap.NewOrderedMap[value.Value, value.Value]()
	// Table.change
	tableModule.Set(
		value.TableKey{Value: "change"},
		value.BuiltinFunction{
			Contract: value.BuiltinFunctionContract{
				Parameters: ordmap.OrderedMapFromArgs([]ordmap.KV[value.TableKey, value.Value]{
					{Key: value.TableKey{Value: "table"}, Value: nil},
					{Key: value.TableKey{Value: "changes"}, Value: nil},
				}),
				Rest: nil,
			},
			Fn: func(args map[string]value.Value) value.Value {
				table := args["table"]
				switch table := table.(type) {
				case value.Table:
				default:
					panic(fmt.Sprintf("first argument to table.change must be a Table, not a %T", table))
				}
				var newTable value.Table
				deepcopy.Copy(&newTable, table)
				changes := args["changes"]

				switch changes := changes.(type) {
				case value.Table:
				default:
					panic(fmt.Sprintf("second argument to table.change must be a Table, not a %T", changes))
				}

				checkedChanges := changes.(value.Table)

				for _, key := range checkedChanges.Entries.Keys() {
					value, _ := checkedChanges.Entries.Get(key)
					newTable.Entries.Set(key, value)
				}

				return newTable
			},
		},
	)
	// Table.delete
	tableModule.Set(
		value.TableKey{Value: "delete"},
		value.BuiltinFunction{
			Contract: value.BuiltinFunctionContract{
				Parameters: ordmap.OrderedMapFromArgs([]ordmap.KV[value.TableKey, value.Value]{
					{Key: value.TableKey{Value: "table"}, Value: nil},
					{Key: value.TableKey{Value: "index"}, Value: nil},
				}),
				Rest: nil,
			},
			Fn: func(args map[string]value.Value) value.Value {
				oldTable := args["table"].(value.Table)
				newTable := value.Table{Entries: orderedmap.NewOrderedMap[value.Value, value.Value]()}
				ind := 0
				for _, key := range oldTable.Entries.Keys() {
					entryValue, _ := oldTable.Entries.Get(key)
					if key == args["index"] {
						continue
					}
					if key.Type() == types.NUMBER {
						newTable.Entries.Set(value.Number{Value: float64(ind)}, entryValue)
						ind += 1
					} else {
						newTable.Entries.Set(key, entryValue)
					}
				}

				return newTable
			},
		},
	)

	// Program module
	programModule := orderedmap.NewOrderedMap[value.Value, value.Value]()
	// Program.crash
	programModule.Set(
		value.TableKey{Value: "crash"},
		value.BuiltinFunction{
			Contract: value.BuiltinFunctionContract{
				Parameters: ordmap.OrderedMapFromArgs([]ordmap.KV[value.TableKey, value.Value]{
					{Key: value.TableKey{Value: "reason"}, Value: nil},
					{Key: value.TableKey{Value: "exit_code"}, Value: value.Number{Value: 1}},
				}),
				Rest: nil,
			},
			Fn: func(args map[string]value.Value) value.Value {
				fmt.Printf("Crash: %s\n", args["reason"].Inspect())
				os.Exit(int(args["exit_code"].(value.Number).Value))
				return nil
			},
		},
	)

	// Error module
	errorModule := orderedmap.NewOrderedMap[value.Value, value.Value]()
	// Error.error
	errorModule.Set(
		value.TableKey{Value: "error"},
		value.BuiltinFunction{
			Contract: value.BuiltinFunctionContract{
				Parameters: ordmap.OrderedMapFromArgs([]ordmap.KV[value.TableKey, value.Value]{
					{Key: value.TableKey{Value: "message"}, Value: nil},
				}),
				Rest: nil,
			},
			Fn: func(args map[string]value.Value) value.Value {
				message := args["message"]
				if message.Type() != types.TEXT {
					panic("you need to supply text to Errors.error")
				}
				return value.Error{Value: message.(value.Text).Value}
			},
		},
	)

	// Type module
	typeModule := orderedmap.NewOrderedMap[value.Value, value.Value]()
	// Type.number
	typeModule.Set(value.TableKey{Value: "number"}, value.Type{Value: types.NUMBER})
	// Type.boolean
	typeModule.Set(value.TableKey{Value: "boolean"}, value.Type{Value: types.BOOL})
	// Type.text
	typeModule.Set(value.TableKey{Value: "text"}, value.Type{Value: types.TEXT})
	// Type.function
	typeModule.Set(value.TableKey{Value: "function"}, value.Type{Value: types.FUNCTION})
	// Type.table
	typeModule.Set(value.TableKey{Value: "table"}, value.Type{Value: types.TABLE})
	// Type.error
	typeModule.Set(value.TableKey{Value: "error"}, value.Type{Value: types.ERROR})
	// Type.type
	typeModule.Set(
		value.TableKey{Value: "type"},
		value.BuiltinFunction{
			Contract: value.BuiltinFunctionContract{
				Parameters: ordmap.OrderedMapFromArgs([]ordmap.KV[value.TableKey, value.Value]{
					{Key: value.TableKey{Value: "value"}, Value: nil},
				}),
				Rest: nil,
			},
			Fn: func(args map[string]value.Value) value.Value {
				switch args["value"].(type) {
				case value.Number:
					return value.Type{Value: types.NUMBER}
				case value.Boolean:
					return value.Type{Value: types.BOOL}
				case value.Text:
					return value.Type{Value: types.TEXT}
				case value.Function:
					return value.Type{Value: types.FUNCTION}
				case value.BuiltinFunction:
					return value.Type{Value: types.FUNCTION}
				case value.Table:
					return value.Type{Value: types.TABLE}
				case value.Error:
					return value.Type{Value: types.ERROR}
				case value.Type:
					return value.Type{Value: types.TYPE}
				default:
					return value.Error{Value: "unknown type"}
				}
			},
		},
	)

	// Text module
	textModule := orderedmap.NewOrderedMap[value.Value, value.Value]()
	// Text.split
	textModule.Set(
		value.TableKey{Value: "split"},
		value.BuiltinFunction{
			Contract: value.BuiltinFunctionContract{
				Parameters: ordmap.OrderedMapFromArgs([]ordmap.KV[value.TableKey, value.Value]{
					{Key: value.TableKey{Value: "text"}, Value: nil},
					{Key: value.TableKey{Value: "separator"}, Value: nil},
				}),
				Rest: nil,
			},
			Fn: func(args map[string]value.Value) value.Value {
				tbl := value.Table{Entries: orderedmap.NewOrderedMap[value.Value, value.Value]()}
				split := strings.Split(args["text"].(value.Text).Value, args["separator"].(value.Text).Value)
				for i, s := range split {
					tbl.Entries.Set(value.Number{Value: float64(i)}, value.Text{Value: s})
				}
				fmt.Println("text split", tbl.Inspect())
				return tbl
			},
		},
	)

	builtinLib.Set(ast.Identifier{Value: "IO"}, ioModule)
	builtinLib.Set(ast.Identifier{Value: "Table"}, tableModule)
	builtinLib.Set(ast.Identifier{Value: "Program"}, programModule)
	builtinLib.Set(ast.Identifier{Value: "Error"}, errorModule)
	builtinLib.Set(ast.Identifier{Value: "Type"}, typeModule)
	builtinLib.Set(ast.Identifier{Value: "Text"}, textModule)

	return builtinLib
}
