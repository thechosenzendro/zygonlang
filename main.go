package main

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"unicode"

	"github.com/davecgh/go-spew/spew"
)

func main() {
	sourceCode, err := os.ReadFile("./main.zygon")
	if err != nil {
		log.Fatal(err)
	}
	// the lexer needs to lex indents correctly
	tokens := tokenize(string(sourceCode) + "\n")
	log.Println(tokens)

	ast := parse(&tokens)
	spew.Dump(ast)

	env := &Environment{Store: make(map[string]Value), Outer: nil}

	res := eval(ast, env)
	if res != nil {
		log.Println(res.Inspect())
	}
}

type Stream[T comparable] struct {
	Index    int
	Contents []T
}

func (stream Stream[T]) peek(amount int) *T {
	index := stream.Index + amount

	if index >= len(stream.Contents) {
		return nil
	}
	return &stream.Contents[index]
}

func (stream *Stream[T]) consume(amount int) {
	stream.Index += amount
}

func isToken(tokens *Stream[Token], tokenType TokenType, amount int) bool {
	index := tokens.Index + amount

	if index >= len(tokens.Contents) {
		return false
	}
	return tokens.Contents[index].Type == tokenType
}

type TokenType string

const (
	UNKNOWN      = "UNKNOWN"
	EOF          = "EOF"
	IDENT        = "IDENT"
	NUM          = "NUM"
	COLON        = "COLON"
	LPAREN       = "LPAREN"
	RPAREN       = "RPAREN"
	COMMA        = "COMMA"
	PLUS         = "PLUS"
	INDENT       = "INDENT"
	DEDENT       = "DEDENT"
	LBRACE       = "LBRACE"
	RBRACE       = "RBRACE"
	CASE         = "CASE"
	TEXT_START   = "TEXT_START"
	TEXT_PART    = "TEXT_PART"
	TEXT_END     = "TEXT_END"
	MINUS        = "MINUS"
	STAR         = "STAR"
	SLASH        = "SLASH"
	IS           = "IS"
	IS_NOT       = "IS_NOT"
	NOT          = "NOT"
	AND          = "AND"
	OR           = "OR"
	LESSER_THAN  = "LESSER_THAN"
	GREATER_THAN = "GREATER_THAN"
	EOL          = "EOL"
	PUB          = "PUB"
	USING        = "USING"
	TRUE         = "TRUE"
	FALSE        = "FALSE"
)

type Token struct {
	Type  TokenType
	Value string
}

var parenLevel = 0
var braceLevel = 0
var indentLevel = []int{0}

func tokenize(sourceCode string) Stream[Token] {
	source := &Stream[rune]{0, []rune(sourceCode)}
	tokens := Stream[Token]{0, []Token{}}

	for source.peek(0) != nil {
		tokens.Contents = append(tokens.Contents, lexToken(source)...)
	}
	tokens.Contents = append(tokens.Contents, Token{EOF, ""})
	return tokens
}

func lexToken(source *Stream[rune]) []Token {
	tokens := []Token{}

	switch {

	case *source.peek(0) == '#':
		for *source.peek(0) != '\n' {
			source.consume(1)
		}
	case unicode.IsLetter(*source.peek(0)) || *source.peek(0) == '_':
		buf := []rune{}
		for source.peek(0) != nil && (unicode.IsLetter(*source.peek(0)) || *source.peek(0) == '_' || *source.peek(0) == '.') {
			buf = append(buf, *source.peek(0))
			source.consume(1)
		}
		if string(buf) == "case" {
			tokens = append(tokens, Token{CASE, "case"})
		} else if string(buf) == "is" {
			tokens = append(tokens, Token{IS, "is"})
		} else if string(buf) == "not" {
			tokens = append(tokens, Token{NOT, "not"})
		} else if string(buf) == "and" {
			tokens = append(tokens, Token{AND, "and"})
		} else if string(buf) == "or" {
			tokens = append(tokens, Token{OR, "or"})
		} else if string(buf) == "pub" {
			tokens = append(tokens, Token{PUB, "pub"})
		} else if string(buf) == "using" {
			tokens = append(tokens, Token{USING, "using"})
		} else if string(buf) == "true" {
			tokens = append(tokens, Token{TRUE, "true"})
		} else if string(buf) == "false" {
			tokens = append(tokens, Token{FALSE, "false"})
		} else {
			tokens = append(tokens, Token{IDENT, string(buf)})

		}

	case unicode.IsDigit(*source.peek(0)):
		buf := []rune{}
		hasDecimal := false
		for source.peek(0) != nil && (unicode.IsDigit(*source.peek(0)) || *source.peek(0) == '_' || *source.peek(0) == '.') {
			if *source.peek(0) == '.' {
				if hasDecimal {
					log.Fatal("Number literal cannot have more decimal parts")
				} else {
					hasDecimal = true
				}
			}
			buf = append(buf, *source.peek(0))
			source.consume(1)
		}
		if buf[len(buf)-1] == '.' {
			log.Fatal("No fractional part")
		}
		tokens = append(tokens, Token{NUM, string(buf)})

	case *source.peek(0) == '"':
		source.consume(1)
		tokens = append(tokens, Token{TEXT_START, ""})
		buf := []rune{}
		for {
			if *source.peek(0) == '"' {
				break
			} else if *source.peek(0) == '{' {
				braceLevel += 1
				bl := braceLevel
				tokens = append(tokens, Token{TEXT_PART, string(buf)})
				buf = []rune{}
				source.consume(1)
				for *source.peek(0) != '}' && braceLevel == bl {
					tokens = append(tokens, lexToken(source)...)
				}
				source.consume(1)
				braceLevel -= 1

			} else if *source.peek(0) == '\\' {
				source.consume(1)
				buf = append(buf, *source.peek(0))
				source.consume(1)
			} else {
				buf = append(buf, *source.peek(0))
				source.consume(1)
			}
		}
		source.consume(1)
		tokens = append(tokens, Token{TEXT_PART, string(buf)})
		tokens = append(tokens, Token{TEXT_END, ""})

	case *source.peek(0) != '\n' && unicode.IsSpace(*source.peek(0)):
		source.consume(1)

	case *source.peek(0) == '\n':
		source.consume(1)
		tokens = append(tokens, Token{EOL, "\\n"})
		currentIndentLevel := 0

		if source.peek(0) != nil {
			for {
				if *source.peek(0) == ' ' {
					currentIndentLevel += 1
					source.consume(1)

				} else if *source.peek(0) == '\t' {
					currentIndentLevel += 4
					source.consume(1)

				} else {
					break
				}
			}
		}
		for {
			if currentIndentLevel == indentLevel[len(indentLevel)-1] {
				break
			}
			if currentIndentLevel > indentLevel[len(indentLevel)-1] {
				indentLevel = append(indentLevel, currentIndentLevel)
				tokens = append(tokens, Token{INDENT, strconv.Itoa(currentIndentLevel)})
			} else if currentIndentLevel < indentLevel[len(indentLevel)-1] {
				tokens = append(tokens, Token{DEDENT, strconv.Itoa(indentLevel[len(indentLevel)-1])})
				indentLevel = indentLevel[:len(indentLevel)-1]
			}
		}

	case *source.peek(0) == '(':
		parenLevel += 1
		tokens = append(tokens, Token{LPAREN, strconv.Itoa(parenLevel)})
		source.consume(1)

	case *source.peek(0) == ')':
		parenLevel -= 1
		tokens = append(tokens, Token{RPAREN, strconv.Itoa(parenLevel + 1)})
		source.consume(1)

	case *source.peek(0) == '{':
		braceLevel += 1
		tokens = append(tokens, Token{LBRACE, strconv.Itoa(braceLevel)})
		source.consume(1)

	case *source.peek(0) == '}':
		braceLevel -= 1
		tokens = append(tokens, Token{RBRACE, strconv.Itoa(braceLevel + 1)})
		source.consume(1)

	case *source.peek(0) == ',':
		tokens = append(tokens, Token{COMMA, ","})
		source.consume(1)

	case *source.peek(0) == '+':
		tokens = append(tokens, Token{PLUS, "+"})
		source.consume(1)

	case *source.peek(0) == '-':
		tokens = append(tokens, Token{MINUS, "-"})
		source.consume(1)

	case *source.peek(0) == '*':
		tokens = append(tokens, Token{STAR, "*"})
		source.consume(1)

	case *source.peek(0) == '/':
		tokens = append(tokens, Token{SLASH, "/"})
		source.consume(1)

	case *source.peek(0) == ':':
		tokens = append(tokens, Token{COLON, ":"})
		source.consume(1)
	case *source.peek(0) == '<':
		tokens = append(tokens, Token{LESSER_THAN, "<"})
		source.consume(1)
	case *source.peek(0) == '>':
		tokens = append(tokens, Token{GREATER_THAN, ">"})
		source.consume(1)
	default:
		tokens = append(tokens, Token{UNKNOWN, string(*source.peek(0))})
		source.consume(1)
	}
	return tokens
}

type Node interface{}

type Expression interface {
	Node
	Expr()
}

type Statement interface {
	Node
	Stmt()
}

type Program struct {
	Body []Node
}

type Identifier struct {
	Value []string
}

func (Identifier) Expr() {}

type NumberLiteral struct {
	Value float64
}

func (NumberLiteral) Expr() {}

type BooleanLiteral struct {
	Value bool
}

func (BooleanLiteral) Expr() {}

type TextLiteral struct {
	Parts []Expression
}

func (TextLiteral) Expr() {}

type TextPart struct {
	Value string
}

func (s TextPart) Expr() {}

type PubStatement struct {
	Public Node
}

func (PubStatement) Stmt() {}

type AssignmentStatement struct {
	Name  Identifier
	Value Block
}

func (AssignmentStatement) Stmt() {}

type CaseExpression struct {
	Cases []struct {
		Pattern Expression
		Block   Block
	}
}

func (CaseExpression) Expr() {}

type Block struct {
	Body []Expression
}

type UsingStatement struct {
	Modules []struct {
		Module  Identifier
		Symbols []Identifier
	}
}

func (UsingStatement) Stmt() {}

type PrefixExpression struct {
	Operator string
	Right    Expression
}

func (PrefixExpression) Expr() {}

type InfixExpression struct {
	Left     Expression
	Operator string
	Right    Expression
}

func (InfixExpression) Expr() {}

type FunctionDeclaration struct {
	Name       *Identifier
	Parameters []struct {
		Name    Identifier
		Default Expression
	}
	Body Block
}

func (FunctionDeclaration) Expr() {}

type FunctionCall struct {
	Fn        Expression
	Arguments []struct {
		Name  *Identifier
		Value Expression
	}
}

func (FunctionCall) Expr() {}

type TableLiteral struct {
	Entries []struct {
		Key   *Identifier
		Value Expression
	}
}

func (TableLiteral) Expr() {}

type (
	prefixParseFn func(*Stream[Token]) Expression
	infixParseFn  func(*Stream[Token], Expression) Expression
)

var prefixParseFns = map[TokenType]prefixParseFn{}
var infixParseFns = map[TokenType]infixParseFn{}
var precedences = map[TokenType]int{
	IS:           EQUALS,
	LESSER_THAN:  LESSGREATER,
	GREATER_THAN: LESSGREATER,
	PLUS:         SUM,
	MINUS:        SUM,
	STAR:         PRODUCT,
	SLASH:        PRODUCT,
	AND:          ANDPREC,
	OR:           ORPREC,
	LPAREN:       CALL,
}

func getPrecedence(token Token) int {
	if precedence, ok := precedences[token.Type]; ok {
		return precedence
	}
	return LOWEST
}

func parseGroupedExpression(tokens *Stream[Token]) Expression {
	tokens.consume(1)
	expr := parseExpression(tokens, LOWEST)
	if tokens.peek(1).Type != RPAREN {
		log.Fatal("no rparen")
	}
	tokens.consume(1)
	return expr
}

func parse(tokens *Stream[Token]) Program {
	program := Program{Body: []Node{}}
	prefixParseFns[IDENT] = parseIdentifier
	prefixParseFns[NUM] = parseNumberLiteral
	prefixParseFns[NOT] = parsePrefixExpression
	prefixParseFns[MINUS] = parsePrefixExpression
	prefixParseFns[TRUE] = parseBooleanLiteral
	prefixParseFns[FALSE] = parseBooleanLiteral
	prefixParseFns[LPAREN] = resolveLParen
	prefixParseFns[CASE] = parseCaseExpression
	prefixParseFns[TEXT_START] = parseTextLiteral
	prefixParseFns[LBRACE] = parseTableLiteral

	infixParseFns[PLUS] = parseInfixExpression
	infixParseFns[MINUS] = parseInfixExpression
	infixParseFns[STAR] = parseInfixExpression
	infixParseFns[SLASH] = parseInfixExpression
	infixParseFns[IS] = parseIsExpression
	infixParseFns[GREATER_THAN] = parseInfixExpression
	infixParseFns[LESSER_THAN] = parseInfixExpression
	infixParseFns[AND] = parseInfixExpression
	infixParseFns[OR] = parseInfixExpression
	infixParseFns[LPAREN] = parseFunction
	for tokens.peek(0).Type != EOF {
		if tokens.peek(0).Type != EOL {
			var node Node = nil
			if isToken(tokens, IDENT, 0) && isToken(tokens, COLON, 1) {
				node = parseAssignmentStatement(tokens)
			} else if isToken(tokens, USING, 0) {
				node = parseUsingStatement(tokens)
			} else if isToken(tokens, PUB, 0) {
				node = parsePubStatement(tokens)
			} else {
				node = parseExpression(tokens, LOWEST)
			}
			program.Body = append(program.Body, node)
		}
		tokens.consume(1)
	}
	return program
}

const (
	_ int = iota
	LOWEST
	ORPREC
	ANDPREC
	EQUALS
	LESSGREATER
	SUM
	PRODUCT
	PREFIX
	CALL
)

func resolveLParen(tokens *Stream[Token]) Expression {
	i := 0
	parenLevel := tokens.peek(0).Value

	for !(isToken(tokens, RPAREN, i) && tokens.peek(i).Value == parenLevel) {
		i += 1
	}
	i += 1
	if isToken(tokens, COLON, i) {
		return parseFunction(tokens, nil)
	} else {
		return parseGroupedExpression(tokens)
	}
}

func parseFunction(tokens *Stream[Token], fn Expression) Expression {
	isDeclaration := false
	if fn == nil {
		isDeclaration = true
	} else {
		i := 0
		parenLevel := tokens.peek(0).Value

		for !(isToken(tokens, RPAREN, i) && tokens.peek(i).Value == parenLevel) {
			i += 1
		}
		i += 1
		if isToken(tokens, COLON, i) {
			isDeclaration = true
		}
	}
	if isDeclaration {
		expr := FunctionDeclaration{}
		if fn != nil {
			switch fn := fn.(type) {
			case Identifier:
				expr.Name = &fn
			}
		} else {
			expr.Name = nil
		}
		parenLevel := tokens.peek(0).Value
		tokens.consume(1)
		for !(isToken(tokens, RPAREN, 0) && tokens.peek(0).Value == parenLevel) {
			if isToken(tokens, IDENT, 0) {
				parameter := struct {
					Name    Identifier
					Default Expression
				}{}
				parameter.Name = parseIdentifier(tokens).(Identifier)
				tokens.consume(1)
				if isToken(tokens, COLON, 0) {
					tokens.consume(1)
					parameter.Default = parseExpression(tokens, LOWEST)
					tokens.consume(1)
				}
				expr.Parameters = append(expr.Parameters, parameter)
			} else if isToken(tokens, COMMA, 0) {
				tokens.consume(1)
			}
		}
		tokens.consume(1)
		if !isToken(tokens, COLON, 0) {
			log.Fatal("no colon in function declaration")
		}
		tokens.consume(1)
		expr.Body = parseBlock(tokens)
		return expr
	} else {
		expr := FunctionCall{}
		expr.Fn = fn

		parenLevel := tokens.peek(0).Value
		tokens.consume(1)
		for !(isToken(tokens, RPAREN, 0) && tokens.peek(0).Value == parenLevel) {
			if isToken(tokens, IDENT, 0) {
				argument := struct {
					Name  *Identifier
					Value Expression
				}{}
				switch name := parseIdentifier(tokens).(type) {
				case Identifier:
					argument.Name = &name
				}
				tokens.consume(1)
				if isToken(tokens, COLON, 0) {
					tokens.consume(1)
					argument.Value = parseExpression(tokens, LOWEST)
					tokens.consume(1)
				}
				expr.Arguments = append(expr.Arguments, argument)
			} else if isToken(tokens, COMMA, 0) {
				tokens.consume(1)
			} else {
				argument := struct {
					Name  *Identifier
					Value Expression
				}{}
				argument.Name = nil
				argument.Value = parseExpression(tokens, LOWEST)
				tokens.consume(1)
				expr.Arguments = append(expr.Arguments, argument)
			}
		}
		return expr
	}
}

func parseAssignmentStatement(tokens *Stream[Token]) Statement {
	stmt := AssignmentStatement{}
	stmt.Name = parseIdentifier(tokens).(Identifier)

	tokens.consume(1)

	if !isToken(tokens, COLON, 0) {
		log.Fatal("no colon in assignment statement")
	}

	tokens.consume(1)

	stmt.Value = parseBlock(tokens)

	return stmt
}

func parsePubStatement(tokens *Stream[Token]) Statement {
	stmt := &PubStatement{}
	tokens.consume(1)
	if isToken(tokens, IDENT, 0) && isToken(tokens, COLON, 1) {
		stmt.Public = parseAssignmentStatement(tokens)
	} else {
		stmt.Public = parseExpression(tokens, LOWEST)
	}
	return stmt
}

func parseUsingStatement(tokens *Stream[Token]) Statement {
	tokens.consume(1)
	stmt := &UsingStatement{}
	for {
		if isToken(tokens, EOL, 0) {
			break
		} else if isToken(tokens, IDENT, 0) {
			mod := struct {
				Module  Identifier
				Symbols []Identifier
			}{Module: parseIdentifier(tokens).(Identifier)}

			tokens.consume(1)

			// means ident ends with a dot
			if mod.Module.Value[len(mod.Module.Value)-1] == "" {
				if !isToken(tokens, LPAREN, 0) {
					log.Fatal("no left paren in using")
				}
				tokens.consume(1)
				for {
					if isToken(tokens, RPAREN, 0) {
						break
					} else if isToken(tokens, IDENT, 0) {
						mod.Symbols = append(mod.Symbols, parseIdentifier(tokens).(Identifier))
						tokens.consume(1)
					} else if isToken(tokens, COMMA, 0) {
						tokens.consume(1)
					}
				}
				tokens.consume(1)
			}

			stmt.Modules = append(stmt.Modules, mod)
		} else if isToken(tokens, COMMA, 0) {
			tokens.consume(1)
		}

	}
	return stmt
}

func parseTableLiteral(tokens *Stream[Token]) Expression {
	expr := TableLiteral{}
	braceLevel := tokens.peek(0).Value
	tokens.consume(1)
	for !(isToken(tokens, RBRACE, 0) && tokens.peek(0).Value == braceLevel) {
		entry := struct {
			Key   *Identifier
			Value Expression
		}{}
		if isToken(tokens, IDENT, 0) {
			val := parseIdentifier(tokens)

			tokens.consume(1)

			if isToken(tokens, COLON, 0) {
				switch key := val.(type) {
				case Identifier:
					entry.Key = &key
				}
				tokens.consume(1)
				entry.Value = parseExpression(tokens, LOWEST)
				tokens.consume(1)
				expr.Entries = append(expr.Entries, entry)
			} else {
				entry.Key = nil
				entry.Value = val
				expr.Entries = append(expr.Entries, entry)
			}
		} else if isToken(tokens, COMMA, 0) {
			tokens.consume(1)
		} else if isToken(tokens, EOL, 0) || isToken(tokens, INDENT, 0) || isToken(tokens, DEDENT, 0) {
			tokens.consume(1)
		} else {
			entry.Key = nil
			entry.Value = parseExpression(tokens, LOWEST)
			tokens.consume(1)

			expr.Entries = append(expr.Entries, entry)
		}
	}
	return expr
}

func parseCaseExpression(tokens *Stream[Token]) Expression {
	expr := CaseExpression{}
	tokens.consume(1)
	if !isToken(tokens, COLON, 0) {
		log.Fatal("no colon in case expression")
	}
	tokens.consume(1)
	if !isToken(tokens, EOL, 0) {
		log.Fatal("no newline in case expression")
	}
	tokens.consume(1)

	if !isToken(tokens, INDENT, 0) {
		log.Fatal("case expression must contain indentation")
	}
	indentLevel := tokens.peek(0).Value
	tokens.consume(1)
	for {
		tok := tokens.peek(0)
		if tok.Type == DEDENT && tok.Value == indentLevel {
			break
		}
		pattern := parseExpression(tokens, LOWEST)
		tokens.consume(1)
		if !isToken(tokens, COLON, 0) {
			log.Fatal("no colon in case expression")
		}
		tokens.consume(1)
		block := parseBlock(tokens)
		tokens.consume(1)
		if isToken(tokens, EOL, 0) {
			tokens.consume(1)
		}
		expr.Cases = append(expr.Cases, struct {
			Pattern Expression
			Block   Block
		}{pattern, block})
	}
	return expr
}

func parseBlock(tokens *Stream[Token]) Block {
	block := Block{}
	if !isToken(tokens, EOL, 0) {
		expr := parseExpression(tokens, LOWEST)
		block.Body = []Expression{expr}
		return block
	}
	tokens.consume(1)
	if !isToken(tokens, INDENT, 0) {
		log.Fatal("no indent in case expression")
	}
	indentLevel := tokens.peek(0).Value
	tokens.consume(1)
	for {
		tok := tokens.peek(0)
		if tok.Type == EOL {
			tokens.consume(1)
		}
		tok = tokens.peek(0)

		if tok.Type == DEDENT && tok.Value == indentLevel {
			break
		}
		block.Body = append(block.Body, parseExpression(tokens, LOWEST))
		tokens.consume(1)
	}
	return block
}

func parseBooleanLiteral(tokens *Stream[Token]) Expression {
	if isToken(tokens, TRUE, 0) {
		return BooleanLiteral{true}
	} else {
		return BooleanLiteral{false}
	}
}

func parseInfixExpression(tokens *Stream[Token], left Expression) Expression {
	expr := InfixExpression{Left: left, Operator: string(tokens.peek(0).Type)}
	precedence := getPrecedence(*tokens.peek(0))
	tokens.consume(1)
	expr.Right = parseExpression(tokens, precedence)
	return expr
}

func parseIsExpression(tokens *Stream[Token], left Expression) Expression {
	expr := InfixExpression{Left: left, Operator: IS}
	precedence := getPrecedence(*tokens.peek(0))
	tokens.consume(1)
	if isToken(tokens, NOT, 0) {
		expr.Operator = IS_NOT
		tokens.consume(1)
	}
	expr.Right = parseExpression(tokens, precedence)
	return expr
}

func parsePrefixExpression(tokens *Stream[Token]) Expression {
	expr := PrefixExpression{Operator: string(tokens.peek(0).Type)}
	tokens.consume(1)
	expr.Right = parseExpression(tokens, PREFIX)
	return expr
}

func parseNumberLiteral(tokens *Stream[Token]) Expression {
	num, err := strconv.ParseFloat(tokens.peek(0).Value, 64)
	if err != nil {
		log.Fatal(err)
	}
	return NumberLiteral{num}
}

func parseIdentifier(tokens *Stream[Token]) Expression {
	return Identifier{strings.Split(tokens.peek(0).Value, ".")}
}

func parseTextLiteral(tokens *Stream[Token]) Expression {
	tokens.consume(1)
	expr := TextLiteral{}
	for {
		if isToken(tokens, TEXT_END, 0) {
			break
		} else if isToken(tokens, TEXT_PART, 0) {
			expr.Parts = append(expr.Parts, TextPart{tokens.peek(0).Value})
			tokens.consume(1)
		} else {
			expr.Parts = append(expr.Parts, parseExpression(tokens, LOWEST))
			tokens.consume(1)
		}
	}
	return expr
}

func parseExpression(tokens *Stream[Token], precedence int) Expression {
	prefix := prefixParseFns[tokens.peek(0).Type]
	if prefix == nil {
		log.Fatalf("No prefix parser for %s", tokens.peek(0))
	}
	leftExpr := prefix(tokens)

	for !(tokens.peek(1).Type == EOL) && precedence < getPrecedence(*tokens.peek(1)) {
		infix := infixParseFns[tokens.peek(1).Type]
		if infix == nil {
			return leftExpr
		}
		tokens.consume(1)
		leftExpr = infix(tokens, leftExpr)
	}

	return leftExpr
}

type Value interface {
	Type() string
	Inspect() string
}

const (
	NUMBER   = "Number"
	BOOL     = "Boolean"
	TEXT     = "Text"
	FUNCTION = "Function"
)

type Number struct {
	Value float64
}

func (n Number) Type() string    { return NUMBER }
func (n Number) Inspect() string { return fmt.Sprintf("%f", n.Value) }

type Boolean struct {
	Value bool
}

func (b Boolean) Type() string    { return BOOL }
func (b Boolean) Inspect() string { return fmt.Sprintf("%t", b.Value) }

type Text struct {
	Value string
}

func (s Text) Type() string    { return TEXT }
func (s Text) Inspect() string { return s.Value }

type Function struct {
	Parameters []struct {
		Name    Identifier
		Default Value
	}
	Body Block
	env  *Environment
}

func (f Function) Type() string    { return FUNCTION }
func (f Function) Inspect() string { return "Function Declaration" }

type Environment struct {
	Store map[string]Value
	Outer *Environment
}

func (e *Environment) Get(name string) (Value, bool) {
	obj, ok := e.Store[name]
	if !ok && e.Outer != nil {
		obj, ok = e.Outer.Get(name)
	}
	return obj, ok
}

func (e *Environment) Set(name string, val Value) Value {
	e.Store[name] = val
	return val
}

func eval(_node Node, env *Environment) Value {
	switch node := _node.(type) {
	case Program:
		var res Value
		for _, nd := range node.Body {
			res = eval(nd, env)
		}
		return res
	case NumberLiteral:
		return Number(node)
	case BooleanLiteral:
		return Boolean(node)
	case TextLiteral:
		str := ""
		for _, part := range node.Parts {
			switch part := part.(type) {
			case TextPart:
				str = str + part.Value
			default:
				str = str + eval(part, env).Inspect()
			}
		}
		return Text{str}
	case PrefixExpression:
		right := eval(node.Right, env)
		switch node.Operator {
		case NOT:
			switch right := right.(type) {
			case Boolean:
				return Boolean{!right.Value}
			default:
				log.Fatal("non boolean passed to not")
			}
		case MINUS:
			switch right := right.(type) {
			case Number:
				return Number{-right.Value}
			}
		}
	case InfixExpression:
		left := eval(node.Left, env)
		right := eval(node.Right, env)
		if left.Type() == NUMBER && right.Type() == NUMBER {
			switch node.Operator {
			case PLUS:
				return Number{left.(Number).Value + right.(Number).Value}
			case MINUS:
				return Number{left.(Number).Value - right.(Number).Value}
			case STAR:
				return Number{left.(Number).Value * right.(Number).Value}
			case SLASH:
				return Number{left.(Number).Value / right.(Number).Value}
			case GREATER_THAN:
				return Boolean{left.(Number).Value > right.(Number).Value}
			case LESSER_THAN:
				return Boolean{left.(Number).Value < right.(Number).Value}

			}
		}
		switch node.Operator {
		case IS:
			return Boolean{left == right}
		case IS_NOT:
			return Boolean{left != right}
		}
	case Block:
		var res Value
		for _, nd := range node.Body {
			res = eval(nd, env)
		}
		return res

	case CaseExpression:
		for _, _case := range node.Cases {
			pattern := eval(_case.Pattern, env)
			if pattern.Type() != BOOL {
				log.Fatal("pattern result is not a boolean")
			}
			if pattern.Inspect() == "true" {
				return eval(_case.Block, env)
			}
		}
		panic("No truthy case in case expr")
	case AssignmentStatement:
		val := eval(node.Value, env)
		env.Set(node.Name.Value[0], val)
		return nil
	case Identifier:
		val, ok := env.Get(node.Value[0])
		if !ok {
			log.Fatal("identifier ", node.Value[0], " not found")
		}
		return val
	case FunctionDeclaration:
		parameters := []struct {
			Name    Identifier
			Default Value
		}{}
		for _, param := range node.Parameters {
			if param.Default == nil {
				parameters = append(parameters, struct {
					Name    Identifier
					Default Value
				}{
					Name:    param.Name,
					Default: nil,
				})
			} else {
				parameters = append(parameters, struct {
					Name    Identifier
					Default Value
				}{
					Name:    param.Name,
					Default: eval(param.Default, env),
				})
			}

		}
		if node.Name == nil {
			return Function{parameters, node.Body, env}
		} else {
			env.Set(node.Name.Value[0], Function{parameters, node.Body, env})
			return nil
		}
	case FunctionCall:
		function := eval(node.Fn, env).(Function)
		e := &Environment{make(map[string]Value), env}

		for i, param := range function.Parameters {
			if len(node.Arguments) > i {
				arg := node.Arguments[i]
				if arg.Name == nil {
					arg.Name = &param.Name
				}
				var value Value = eval(arg.Value, e)
				if arg.Value == nil {
					value = function.Parameters[i].Default
				}
				e.Set(arg.Name.Value[0], value)
			} else {

				name := &param.Name
				value := function.Parameters[i].Default
				if value == nil {
					log.Fatal("no default")
				}
				e.Set(name.Value[0], value)
			}
		}
		return eval(function.Body, e)

	default:
		log.Fatalf("eval error %T", node)
	}
	panic("uhoh")
}
