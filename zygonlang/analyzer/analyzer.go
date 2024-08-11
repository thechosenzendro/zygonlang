package analyzer

import (
	"fmt"
	"reflect"
	"thechosenzendro/zygonlang/zygonlang/ast"
	ordmap "thechosenzendro/zygonlang/zygonlang/orderedmap"
	"thechosenzendro/zygonlang/zygonlang/token"
	"thechosenzendro/zygonlang/zygonlang/types"

	"github.com/elliotchance/orderedmap/v2"
)

func Map[T, V any](ts []T, fn func(T) V) []V {
	result := make([]V, len(ts))
	for i, t := range ts {
		result[i] = fn(t)
	}
	return result
}

func Analyze(program ast.Program) ast.Program {
	program = Typecheck(program)
	return program
}

func Typecheck(program ast.Program) ast.Program {
	typeEnv := &types.TypeEnvironment{
		Store: map[string]*types.Type{},
		Outer: nil,
	}
	for _, node := range program.Body {
		run := true
		switch node := node.(type) {
		case ast.AssignmentStatement:
		case ast.FunctionDeclaration:
		case ast.UsingStatement:
		case ast.PubStatement:
		default:
			run = false
			res := resolveType(node, typeEnv)
			typeEnv.Set("_", res)
		}
		if run {
			resolveType(node, typeEnv)
		}
	}
	return program
}

func resolveType(node ast.Node, typeEnv *types.TypeEnvironment) *types.Type {
	switch node := node.(type) {
	case ast.NumberLiteral:
		return types.NewType(types.NUMBER, nil)
	case ast.BooleanLiteral:
		return types.NewType(types.BOOL, nil)
	case ast.TextLiteral:
		return types.NewType(types.TEXT, nil)
	case ast.PrefixExpression:
		switch node.Operator {
		case token.NOT:
			assert(node.Right, types.NewType(types.BOOL, nil), typeEnv)
			return types.NewType(types.BOOL, nil)
		case token.MINUS:
			assert(node.Right, types.NewType(types.NUMBER, nil), typeEnv)
			return types.NewType(types.NUMBER, nil)
		}
	case ast.InfixExpression:
		op := node.Operator
		switch {
		case op == token.PLUS || op == token.MINUS || op == token.STAR || op == token.SLASH:
			assert(node.Left, types.NewType(types.NUMBER, nil), typeEnv)
			assert(node.Right, types.NewType(types.NUMBER, nil), typeEnv)

			return types.NewType(types.NUMBER, nil)
		case op == token.IS || op == token.IS_NOT:
			leftType := resolveType(node.Left, typeEnv)
			assert(node.Right, leftType, typeEnv)
			return types.NewType(types.BOOL, nil)
		}
	case ast.Block:
		var resType *types.Type
		for _, nd := range node.Body {
			run := true
			switch nd := nd.(type) {
			case ast.AssignmentStatement:
			case ast.FunctionDeclaration:
			case ast.UsingStatement:
			case ast.PubStatement:
			default:
				run = false
				resType = resolveType(nd, typeEnv)
				typeEnv.Set("_", resType)
			}
			if run {
				resolveType(nd, typeEnv)
			}
		}
		return resType
	case ast.CaseExpression:
	case ast.AssignmentStatement:
		typeEnv.Set(node.Name.Value, resolveType(node.Value, typeEnv))
		return nil
	case ast.AccessOperator:
	case ast.Identifier:
		t, _ := typeEnv.Get(node.Value)
		return t
	case ast.FunctionDeclaration:
		funcEnv := &types.TypeEnvironment{
			Store: map[string]*types.Type{},
			Outer: typeEnv,
		}
		params := orderedmap.NewOrderedMap[string, string]()
		for _, key := range node.Parameters.Keys() {
			params.Set(key.Value, "")
			paramDefault, _ := node.Parameters.Get(key)
			if paramDefault != nil {
				funcEnv.Set(key.Value, resolveType(paramDefault, typeEnv))
			} else {
				funcEnv.Set(key.Value, nil)
			}
		}

		retType := resolveType(node.Body, funcEnv)

		t := types.NewType(types.FUNCTION, ordmap.OrderedMapFromArgs([]ordmap.KV[string, *types.Type]{{
			Key:   "?return_type",
			Value: retType,
		}}))
		for _, key := range params.Keys() {
			x, _ := funcEnv.Get(key)
			t.Properties.Set(key, x)
		}

		typeEnv.Set(node.Name.Value, t)

		fmt.Println(t.Inspect(0))
		return t

	case ast.FunctionCall:
	case ast.TableLiteral:
	case ast.PubStatement:
	case ast.UsingStatement:
	case ast.RestOperator:
	}
	panic(fmt.Sprintf("type error %T", node))
}

func assert(node ast.Node, typ *types.Type, typeEnv *types.TypeEnvironment) {
	switch node := node.(type) {
	case ast.Identifier:
		if identType, ok := typeEnv.Get(node.Value); ok {
			if identType == nil {
				typeEnv.Set(node.Value, typ)
			}
		} else {
			panic("identifier not found")
		}
	}
	b := resolveType(node, typeEnv)
	if !reflect.DeepEqual(b, typ) {
		panic(fmt.Sprintf("Bad type\nExpected: %s\nGot: %s", typ.Inspect(0), b.Inspect(0)))
	}

}
