package main

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"reflect"
	"strconv"
	"strings"
	"unicode"

	"github.com/davecgh/go-spew/spew"
	"github.com/elliotchance/orderedmap/v2"
	"github.com/tiendc/go-deepcopy"
)

var projectRoot string

func main() {
	setupBuiltinLib()
	path := "./examples/Main.zygon"
	_base := strings.Split(path, "/")
	projectRoot = strings.Join(_base[:len(_base)-1], "/") + "/"
	sourceCode, err := os.ReadFile("./examples/Main.zygon")
	if err != nil {
		panic(err)
	}

	val, _ := Exec(string(sourceCode))
	if val != nil {
		fmt.Println("Result: ", val.Inspect())
	} else {
		fmt.Println("Result: ", val)
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
	DOT          = "DOT"
	DEFAULT      = "DEFAULT"
	REST         = "REST"
)

type Token struct {
	Type  TokenType
	Value string
}

var parenLevel = 0
var braceLevel = 0
var indentLevel = []int{0}

func Tokenize(sourceCode string) Stream[Token] {
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
		for source.peek(0) != nil && (unicode.IsLetter(*source.peek(0)) || *source.peek(0) == '_') {
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
		} else if string(buf) == "default" {
			tokens = append(tokens, Token{DEFAULT, "default"})
		} else {
			tokens = append(tokens, Token{IDENT, string(buf)})

		}
	case *source.peek(0) == '.':
		if *source.peek(1) == '.' && *source.peek(2) == '.' {
			tokens = append(tokens, Token{REST, "..."})
			source.consume(3)
		} else {
			tokens = append(tokens, Token{DOT, "."})
			source.consume(1)
		}
	case unicode.IsDigit(*source.peek(0)):
		buf := []rune{}
		hasDecimal := false
		for source.peek(0) != nil && (unicode.IsDigit(*source.peek(0)) || *source.peek(0) == '_' || *source.peek(0) == '.') {
			if *source.peek(0) == '.' {
				if hasDecimal {
					panic("Number literal cannot have more decimal parts")
				} else {
					hasDecimal = true
				}
			}
			buf = append(buf, *source.peek(0))
			source.consume(1)
		}
		if buf[len(buf)-1] == '.' {
			panic("No fractional part")
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

type Ident interface {
	Expression
	Ident()
}

type Identifier struct {
	Value string
}

func (Identifier) Expr()  {}
func (Identifier) Ident() {}

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
	Subject Expression
	Cases   []struct {
		Pattern Expression
		Block   Block
	}
	Default *Block
}

func (CaseExpression) Expr() {}

type Block struct {
	Body []Node
}

type UsingStatement struct {
	Modules []struct {
		Module  Ident
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
	Parameters *orderedmap.OrderedMap[Identifier, Expression]
	Rest       *RestOperator
	Body       Block
}

func (FunctionDeclaration) Expr() {}

type FunctionCall struct {
	Fn        Expression
	Arguments []struct {
		Name  *TableKey
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

type Grouped struct {
	Value Expression
}

func (Grouped) Expr() {}

type AccessOperator struct {
	Subject   Expression
	Attribute Expression
}

func (AccessOperator) Expr()  {}
func (AccessOperator) Ident() {}

type RestOperator struct {
	Value Expression
}

func (RestOperator) Expr() {}

type (
	prefixParseFn func(*Stream[Token]) Expression
	infixParseFn  func(*Stream[Token], Expression) Expression
)

var prefixParseFns = map[TokenType]prefixParseFn{}
var infixParseFns = map[TokenType]infixParseFn{}
var precedences = map[TokenType]int{
	DOT:          ACCESS,
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
		panic("no rparen")
	}
	tokens.consume(1)
	return expr
}

func parseRestOperator(tokens *Stream[Token]) Expression {
	expr := RestOperator{}
	if isToken(tokens, RBRACE, 1) || isToken(tokens, RPAREN, 1) || isToken(tokens, EOL, 1) {
		return expr
	}
	tokens.consume(1)
	expr.Value = parseExpression(tokens, LOWEST)
	// ...tokens, ...{}, ...(get(x))
	return expr
}

func Parse(tokens *Stream[Token]) Program {
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
	prefixParseFns[REST] = parseRestOperator

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
	infixParseFns[DOT] = parseAccessOperator
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
	ACCESS
)

func parseAccessOperator(tokens *Stream[Token], left Expression) Expression {
	tokens.consume(1)
	if tokens.peek(0).Type == IDENT {
		return AccessOperator{Subject: left, Attribute: parseIdentifier(tokens)}
	} else if tokens.peek(0).Type == LPAREN {
		return AccessOperator{Subject: left, Attribute: Grouped{parseGroupedExpression(tokens)}}
	}
	panic("Not compatible with index")
}

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

var parsingCase = false

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
		if isToken(tokens, COLON, i) && !parsingCase {
			isDeclaration = true
		}
	}
	if isDeclaration {
		expr := FunctionDeclaration{Parameters: orderedmap.NewOrderedMap[Identifier, Expression]()}
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
				name := parseIdentifier(tokens).(Identifier)
				var param_default Expression = nil
				tokens.consume(1)
				if isToken(tokens, COLON, 0) {
					tokens.consume(1)
					param_default = parseExpression(tokens, LOWEST)
					tokens.consume(1)
				}
				expr.Parameters.Set(name, param_default)
			} else if isToken(tokens, COMMA, 0) {
				tokens.consume(1)
			} else if isToken(tokens, REST, 0) {
				rest := parseRestOperator(tokens).(RestOperator)
				expr.Rest = &rest
				tokens.consume(1)
			}
		}
		tokens.consume(1)
		if !isToken(tokens, COLON, 0) {
			panic("no colon in function declaration")
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
					Name  *TableKey
					Value Expression
				}{}
				if isToken(tokens, COLON, 1) {
					switch name := parseIdentifier(tokens).(type) {
					case Identifier:
						argument.Name = &TableKey{name.Value}
					}
					tokens.consume(2)
				}
				argument.Value = parseExpression(tokens, LOWEST)
				tokens.consume(1)
				expr.Arguments = append(expr.Arguments, argument)
			} else if isToken(tokens, COMMA, 0) {
				tokens.consume(1)
			} else {
				argument := struct {
					Name  *TableKey
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
		panic("no colon in assignment statement")
	}

	tokens.consume(1)

	stmt.Value = parseBlock(tokens)

	return stmt
}

func parsePubStatement(tokens *Stream[Token]) Statement {
	stmt := PubStatement{}
	tokens.consume(1)
	if isToken(tokens, IDENT, 0) && isToken(tokens, COLON, 1) {
		stmt.Public = parseAssignmentStatement(tokens)
	} else {
		stmt.Public = parseExpression(tokens, LOWEST)
	}
	return stmt
}

func parseUsingPath(tokens *Stream[Token]) Ident {
	if isToken(tokens, IDENT, 0) {
		if (isToken(tokens, DOT, 1) && isToken(tokens, LPAREN, 2)) || (isToken(tokens, EOL, 1) || isToken(tokens, COMMA, 1)) {
			return Identifier{tokens.peek(0).Value}
		} else if isToken(tokens, DOT, 1) && isToken(tokens, IDENT, 2) {
			subject := Identifier{tokens.peek(0).Value}
			tokens.consume(2)
			return AccessOperator{Subject: subject, Attribute: parseUsingPath(tokens)}
		}
	}
	panic("a")
}

func parseUsingStatement(tokens *Stream[Token]) Statement {
	tokens.consume(1)
	stmt := UsingStatement{}
	for {
		if isToken(tokens, EOL, 0) {
			break
		} else if isToken(tokens, IDENT, 0) {
			mod := struct {
				Module  Ident
				Symbols []Identifier
			}{Module: parseUsingPath(tokens)}
			fmt.Println(mod.Module)
			tokens.consume(1)
			endsWithDot := false
			if isToken(tokens, DOT, 0) {
				endsWithDot = true
			}

			if endsWithDot {
				tokens.consume(1)
				if !isToken(tokens, LPAREN, 0) {
					panic("no left paren in using")
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
	parsingCase = true
	tokens.consume(1)
	if !isToken(tokens, COLON, 0) {
		expr.Subject = parseExpression(tokens, LOWEST)
		tokens.consume(1)
	}
	if !isToken(tokens, COLON, 0) {
		panic("no colon in case expression")
	}
	tokens.consume(1)
	if !isToken(tokens, EOL, 0) {
		panic("no newline in case expression")
	}
	tokens.consume(1)

	if !isToken(tokens, INDENT, 0) {
		panic("case expression must contain indentation")
	}
	indentLevel := tokens.peek(0).Value
	tokens.consume(1)
	for {
		tok := tokens.peek(0)
		if tok.Type == DEDENT && tok.Value == indentLevel {
			break
		}
		if isToken(tokens, DEFAULT, 0) {
			tokens.consume(1)
			if !isToken(tokens, COLON, 0) {
				panic("no colon in case expression")
			}
			tokens.consume(1)
			block := parseBlock(tokens)
			tokens.consume(1)
			if isToken(tokens, EOL, 0) {
				tokens.consume(1)
			}
			if expr.Default == nil {
				expr.Default = &block
			} else {
				panic("cannot have more than one default in case")
			}
		} else {
			pattern := parseExpression(tokens, LOWEST)
			tokens.consume(1)
			if !isToken(tokens, COLON, 0) {
				panic("no colon in case expression")
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
	}
	parsingCase = false
	return expr
}

func parseBlock(tokens *Stream[Token]) Block {
	block := Block{}
	if !isToken(tokens, EOL, 0) {
		expr := parseExpression(tokens, LOWEST)
		block.Body = []Node{expr}
		return block
	}
	tokens.consume(1)
	if !isToken(tokens, INDENT, 0) {
		panic("no indent in case expression")
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
		panic(err)
	}
	return NumberLiteral{num}
}

func parseIdentifier(tokens *Stream[Token]) Expression {
	return Identifier{tokens.peek(0).Value}
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
		panic(fmt.Sprintf("No prefix parser for %s", tokens.peek(0)))
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
	NUMBER    = "Number"
	BOOL      = "Boolean"
	TEXT      = "Text"
	FUNCTION  = "Function"
	TABLE     = "Table"
	TABLE_KEY = "TableKey"
	BUILTIN   = "BuiltinFunction"
	ERROR     = "Error"
)

type KV[K, V comparable] struct {
	Key   K
	Value V
}

func orderedMapFromArgs[K, V comparable](y []KV[K, V]) *orderedmap.OrderedMap[K, V] {
	x := orderedmap.NewOrderedMap[K, V]()
	for _, z := range y {
		x.Set(z.Key, z.Value)
	}
	return x
}

var builtinLib = orderedmap.NewOrderedMap[Ident, *orderedmap.OrderedMap[Value, Value]]()

func setupBuiltinLib() {
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
			Contract: BuiltinFunctionContract{
				Parameters: orderedMapFromArgs([]KV[TableKey, Value]{
					{TableKey{"prompt"}, nil},
				}),
				Rest: nil,
			},
		},
	)

	// Table module
	tableModule := orderedmap.NewOrderedMap[Value, Value]()
	// Table.change
	tableModule.Set(
		TableKey{"change"},
		BuiltinFunction{
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
			Contract: BuiltinFunctionContract{
				Parameters: orderedMapFromArgs([]KV[TableKey, Value]{
					{TableKey{"table"}, nil},
					{TableKey{"changes"}, nil},
				}),
				Rest: nil,
			},
		},
	)
	// Table.delete
	tableModule.Set(
		TableKey{"delete"},
		BuiltinFunction{
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
			Contract: BuiltinFunctionContract{
				Parameters: orderedMapFromArgs([]KV[TableKey, Value]{
					{TableKey{"table"}, nil},
					{TableKey{"index"}, nil},
				}),
				Rest: nil,
			},
		},
	)

	// Program module
	programModule := orderedmap.NewOrderedMap[Value, Value]()
	// Program.crash
	programModule.Set(
		TableKey{"crash"},
		BuiltinFunction{
			Fn: func(args map[string]Value) Value {
				fmt.Printf("Crash: %s\n", args["reason"].Inspect())
				os.Exit(1)
				return nil
			},
			Contract: BuiltinFunctionContract{
				Parameters: orderedMapFromArgs([]KV[TableKey, Value]{
					{TableKey{"reason"}, nil},
				}),
				Rest: nil,
			},
		},
	)

	// Errors module
	errorsModule := orderedmap.NewOrderedMap[Value, Value]()
	// Errors.error
	errorsModule.Set(
		TableKey{"error"},
		BuiltinFunction{
			Fn: func(args map[string]Value) Value {
				message := args["message"]
				if message.Type() != TEXT {
					panic("you need to supply text to Errors.error")
				}
				return Error{message.(Text).Value}
			},
			Contract: BuiltinFunctionContract{
				Parameters: orderedMapFromArgs([]KV[TableKey, Value]{
					{TableKey{"message"}, nil},
				}),
				Rest: nil,
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
			Contract: BuiltinFunctionContract{
				Parameters: orderedMapFromArgs([]KV[TableKey, Value]{
					{TableKey{"value"}, nil},
				}),
				Rest: nil,
			},
		},
	)

	// Text module
	textModule := orderedmap.NewOrderedMap[Value, Value]()
	// Text.split
	textModule.Set(
		TableKey{"split"},
		BuiltinFunction{
			Fn: func(args map[string]Value) Value {
				tbl := Table{Entries: orderedmap.NewOrderedMap[Value, Value]()}
				split := strings.Split(args["text"].(Text).Value, args["separator"].(Text).Value)
				for i, s := range split {
					tbl.Entries.Set(Number{float64(i)}, Text{s})
				}
				fmt.Println("text split", tbl.Inspect())
				return tbl
			},
			Contract: BuiltinFunctionContract{
				Parameters: orderedMapFromArgs([]KV[TableKey, Value]{
					{TableKey{"text"}, nil},
					{TableKey{"separator"}, nil},
				}),
				Rest: nil,
			},
		},
	)

	builtinLib.Set(Identifier{"IO"}, ioModule)
	builtinLib.Set(Identifier{"Table"}, tableModule)
	builtinLib.Set(Identifier{"Program"}, programModule)
	builtinLib.Set(Identifier{"Errors"}, errorsModule)
	builtinLib.Set(Identifier{"Types"}, typesModule)
	builtinLib.Set(Identifier{"Text"}, textModule)
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
	Body       Block
	Rest       *RestOperator
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
func (t Table) Get(name string) (Value, bool) {
	val, ok := t.Entries.Get(TableKey{name})
	if !ok {
		panic("no ident")
	}
	return val, ok
}

type TableKey struct {
	Value string
}

func (tk TableKey) Type() string    { return TABLE_KEY }
func (tk TableKey) Inspect() string { return tk.Value }

type BuiltinFunctionContract struct {
	Parameters *orderedmap.OrderedMap[TableKey, Value]
	Rest       *RestOperator
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
func (e Error) Inspect() string { return fmt.Sprintf("error(%s)", e.Value) }

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

type Gettable interface {
	Get(name string) (Value, bool)
}

func Eval(node Node, env *Environment) Value {
	switch node := node.(type) {
	case Program:
		var res Value
		for _, nd := range node.Body {
			run := true
			switch nd := nd.(type) {
			case AssignmentStatement:
			case FunctionDeclaration:
			case UsingStatement:
			case PubStatement:
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
				str = str + Eval(part, env).Inspect()
			}
		}
		return Text{str}
	case PrefixExpression:
		right := Eval(node.Right, env)
		switch node.Operator {
		case NOT:
			switch right := right.(type) {
			case Boolean:
				return Boolean{!right.Value}
			default:
				panic("non boolean passed to not")
			}
		case MINUS:
			switch right := right.(type) {
			case Number:
				return Number{-right.Value}
			}
		}
	case InfixExpression:
		switch node.Operator {
		case IS:
			return Boolean{reflect.DeepEqual(Eval(node.Left, env), Eval(node.Right, env))}
		case IS_NOT:
			return Boolean{!reflect.DeepEqual(Eval(node.Left, env), Eval(node.Right, env))}
		case AND:
			return Boolean{Eval(node.Left, env).(Boolean).Value && Eval(node.Right, env).(Boolean).Value}
		case OR:
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

	case Block:
		var res Value
		for _, nd := range node.Body {
			run := true
			switch nd := nd.(type) {
			case AssignmentStatement:
			case FunctionDeclaration:
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

	case CaseExpression:
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
				case TableLiteral:
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
								case Identifier:
									patternEnviron.Set(value.Value, val)
									patternResult = Boolean{true}

								case RestOperator:
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
										patternEnviron.Set(value.Value.(Identifier).Value, table)
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
	case AssignmentStatement:
		if _, ok := env.Get(node.Name.Value); !ok {
			val := Eval(node.Value, env)
			env.Set(node.Name.Value, val)
			return nil
		} else {
			panic(fmt.Sprintf("Cannot reassign identifier %s", node.Name.Value))
		}
	case AccessOperator:
		subject := Eval(node.Subject, env)
		var index Value
		switch attribute := node.Attribute.(type) {
		case Identifier:
			index = TableKey(attribute)
		case Grouped:
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
	case Identifier:
		val, ok := env.Get(node.Value)
		if !ok {
			panic(fmt.Sprintf("identifier %s not found", node.Value))
		}
		return val
	case FunctionDeclaration:
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
	case FunctionCall:
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
					case RestOperator:
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
					case RestOperator:
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
				funcEnviron.Set(function.Rest.Value.(Identifier).Value, rest)
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
					case RestOperator:
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
					case RestOperator:
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
				funcEnviron[function.Contract.Rest.Value.(Identifier).Value] = rest
			}
			return function.Fn(funcEnviron)

		}
	case TableLiteral:
		entries := orderedmap.NewOrderedMap[Value, Value]()
		index := -1
		for _, entry := range node.Entries {
			if entry.Key == nil {
				switch val := entry.Value.(type) {
				case RestOperator:
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
	case PubStatement:
		switch pub := node.Public.(type) {
		case AssignmentStatement:
			Eval(pub, env)
			env.Set("pub "+pub.Name.Value, env.Store[pub.Name.Value])
		case FunctionDeclaration:
			if pub.Name != nil {
				Eval(pub, env)
				env.Set("pub "+pub.Name.Value, env.Store[pub.Name.Value])
			} else {
				panic("anonymous function could not be made public")
			}
		default:
			panic("error in pub statement")
		}
	case UsingStatement:
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
	case RestOperator:
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

func unwrap(m Ident, toPut Table, env *Environment) {
	switch m := m.(type) {
	case AccessOperator:
		unwrap(m.Attribute.(Ident), toPut, env)
	case Identifier:
		env.Set(m.Value, toPut)
	}
}

func getModPath(module Ident) string {
	switch mod := module.(type) {
	case Identifier:
		return "/" + mod.Value + ".zygon"
	case AccessOperator:
		return "/" + mod.Subject.(Identifier).Value + "/" + getModPath(mod.Attribute.(Ident))
	}
	return ""
}

func Exec(sourceCode string) (Value, *Environment) {
	// the lexer needs to lex indents correctly
	tokens := Tokenize(sourceCode + "\n")
	fmt.Println(tokens)

	ast := Parse(&tokens)
	spew.Dump(ast)

	env := &Environment{Store: make(map[string]Value), Outer: nil}

	fmt.Print("\n\n===PROGRAM OUTPUT===\n\n")

	return Eval(ast, env), env
}
