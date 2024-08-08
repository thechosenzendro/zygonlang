package builtin

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"thechosenzendro/zygonlang/zygonlang/evaluator"

	"github.com/elliotchance/orderedmap/v2"
	"github.com/tiendc/go-deepcopy"
)

func BuiltinLib() *orderedmap.OrderedMap[evaluator.Name, *orderedmap.OrderedMap[Value, Value]] {
	builtinLib := orderedmap.NewOrderedMap[Name, *orderedmap.OrderedMap[Value, Value]]()
	// IO module
	ioModule := orderedmap.NewOrderedMap[Value, Value]()
	// IO.log
	ioModule.Set(
		TableKey{"log"},
		BuiltinFunction{
			Contract: BuiltinFunctionContract{
				Parameters: orderedMapFromArgs([]KV[TableKey, Value]{
					{TableKey{"message"}, nil},
				}),
				Rest: nil,
			},

			Fn: func(args map[string]Value) Value {
				fmt.Print(args["message"].Inspect() + "\n")
				return nil
			},
		})
	// IO.get
	ioModule.Set(
		TableKey{"get"},
		BuiltinFunction{
			Contract: BuiltinFunctionContract{
				Parameters: orderedMapFromArgs([]KV[TableKey, Value]{
					{TableKey{"prompt"}, nil},
				}),
				Rest: nil,
			},
			Fn: func(args map[string]Value) Value {
				prompt := args["prompt"].Inspect()
				fmt.Print(prompt)
				var input string
				scanner := bufio.NewScanner(os.Stdin)
				if scanner.Scan() {
					input = scanner.Text()
				}
				return Text{input}
			},
		},
	)

	// Table module
	tableModule := orderedmap.NewOrderedMap[Value, Value]()
	// Table.change
	tableModule.Set(
		TableKey{"change"},
		BuiltinFunction{
			Contract: BuiltinFunctionContract{
				Parameters: orderedMapFromArgs([]KV[TableKey, Value]{
					{TableKey{"table"}, nil},
					{TableKey{"changes"}, nil},
				}),
				Rest: nil,
			},
			Fn: func(args map[string]Value) Value {
				table := args["table"]
				switch table := table.(type) {
				case Table:
				default:
					panic(fmt.Sprintf("first argument to table.change must be a Table, not a %T", table))
				}
				var newTable Table
				deepcopy.Copy(&newTable, table)
				changes := args["changes"]

				switch changes := changes.(type) {
				case Table:
				default:
					panic(fmt.Sprintf("second argument to table.change must be a Table, not a %T", changes))
				}

				checkedChanges := changes.(Table)

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
		TableKey{"delete"},
		BuiltinFunction{
			Contract: BuiltinFunctionContract{
				Parameters: orderedMapFromArgs([]KV[TableKey, Value]{
					{TableKey{"table"}, nil},
					{TableKey{"index"}, nil},
				}),
				Rest: nil,
			},
			Fn: func(args map[string]Value) Value {
				oldTable := args["table"].(Table)
				newTable := Table{Entries: orderedmap.NewOrderedMap[Value, Value]()}
				ind := 0
				for _, key := range oldTable.Entries.Keys() {
					value, _ := oldTable.Entries.Get(key)
					if key == args["index"] {
						continue
					}
					if key.Type() == NUMBER {
						newTable.Entries.Set(Number{float64(ind)}, value)
						ind += 1
					} else {
						newTable.Entries.Set(key, value)
					}
				}

				return newTable
			},
		},
	)

	// Program module
	programModule := orderedmap.NewOrderedMap[Value, Value]()
	// Program.crash
	programModule.Set(
		TableKey{"crash"},
		BuiltinFunction{
			Contract: BuiltinFunctionContract{
				Parameters: orderedMapFromArgs([]KV[TableKey, Value]{
					{TableKey{"reason"}, nil},
					{TableKey{"exit_code"}, Number{1}},
				}),
				Rest: nil,
			},
			Fn: func(args map[string]Value) Value {
				fmt.Printf("Crash: %s\n", args["reason"].Inspect())
				os.Exit(int(args["exit_code"].(Number).Value))
				return nil
			},
		},
	)

	// Errors module
	errorsModule := orderedmap.NewOrderedMap[Value, Value]()
	// Errors.error
	errorsModule.Set(
		TableKey{"error"},
		BuiltinFunction{
			Contract: BuiltinFunctionContract{
				Parameters: orderedMapFromArgs([]KV[TableKey, Value]{
					{TableKey{"message"}, nil},
				}),
				Rest: nil,
			},
			Fn: func(args map[string]Value) Value {
				message := args["message"]
				if message.Type() != TEXT {
					panic("you need to supply text to Errors.error")
				}
				return Error{message.(Text).Value}
			},
		},
	)

	// Types module
	typesModule := orderedmap.NewOrderedMap[Value, Value]()
	// Types.number
	typesModule.Set(TableKey{"number"}, TableKey{NUMBER})
	// Types.boolean
	typesModule.Set(TableKey{"boolean"}, TableKey{BOOL})
	// Types.text
	typesModule.Set(TableKey{"text"}, TableKey{TEXT})
	// Types.function
	typesModule.Set(TableKey{"function"}, TableKey{FUNCTION})
	// Types.table
	typesModule.Set(TableKey{"table"}, TableKey{TABLE})
	// Types.error
	typesModule.Set(TableKey{"error"}, TableKey{ERROR})
	// Types.type
	typesModule.Set(
		TableKey{"type"},
		BuiltinFunction{
			Contract: BuiltinFunctionContract{
				Parameters: orderedMapFromArgs([]KV[TableKey, Value]{
					{TableKey{"value"}, nil},
				}),
				Rest: nil,
			},
			Fn: func(args map[string]Value) Value {
				switch args["value"].(type) {
				case Number:
					return TableKey{NUMBER}
				case Boolean:
					return TableKey{BOOL}
				case Text:
					return TableKey{TEXT}
				case Function:
					return TableKey{FUNCTION}
				case BuiltinFunction:
					return TableKey{FUNCTION}
				case Table:
					return TableKey{TABLE}
				case Error:
					return TableKey{ERROR}
				default:
					return Error{"unknown type"}
				}
			},
		},
	)

	// Text module
	textModule := orderedmap.NewOrderedMap[Value, Value]()
	// Text.split
	textModule.Set(
		TableKey{"split"},
		BuiltinFunction{
			Contract: BuiltinFunctionContract{
				Parameters: orderedMapFromArgs([]KV[TableKey, Value]{
					{TableKey{"text"}, nil},
					{TableKey{"separator"}, nil},
				}),
				Rest: nil,
			},
			Fn: func(args map[string]Value) Value {
				tbl := Table{Entries: orderedmap.NewOrderedMap[Value, Value]()}
				split := strings.Split(args["text"].(Text).Value, args["separator"].(Text).Value)
				for i, s := range split {
					tbl.Entries.Set(Number{float64(i)}, Text{s})
				}
				fmt.Println("text split", tbl.Inspect())
				return tbl
			},
		},
	)

	builtinLib.Set(Identifier{"IO"}, ioModule)
	builtinLib.Set(Identifier{"Table"}, tableModule)
	builtinLib.Set(Identifier{"Program"}, programModule)
	builtinLib.Set(Identifier{"Errors"}, errorsModule)
	builtinLib.Set(Identifier{"Types"}, typesModule)
	builtinLib.Set(Identifier{"Text"}, textModule)

	return builtinLib
}
