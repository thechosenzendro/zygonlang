package main

import (
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
	// the parser needs to parse indents correctly
	tokens := tokenize(string(sourceCode) + "\n")
	log.Println(tokens)

	ast := parse(&tokens)
	spew.Dump(ast)
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
	STRING_START = "STRING_START"
	STRING_PART  = "STRING_PART"
	STRING_END   = "STRING_END"
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

func tokenize(sourceCode string) Stream[Token] {
	source := Stream[rune]{0, []rune(sourceCode)}
	tokens := Stream[Token]{0, []Token{}}

	parenLevel := 0
	braceLevel := 0
	indentLevel := []int{0}
	stringInterpolationOpened := false
	for source.peek(0) != nil {
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
				tokens.Contents = append(tokens.Contents, Token{CASE, "case"})
			} else if string(buf) == "is" {
				tokens.Contents = append(tokens.Contents, Token{IS, "is"})
			} else if string(buf) == "not" {
				tokens.Contents = append(tokens.Contents, Token{NOT, "not"})
			} else if string(buf) == "and" {
				tokens.Contents = append(tokens.Contents, Token{AND, "and"})
			} else if string(buf) == "or" {
				tokens.Contents = append(tokens.Contents, Token{OR, "or"})
			} else if string(buf) == "pub" {
				tokens.Contents = append(tokens.Contents, Token{PUB, "pub"})
			} else if string(buf) == "using" {
				tokens.Contents = append(tokens.Contents, Token{USING, "using"})
			} else if string(buf) == "true" {
				tokens.Contents = append(tokens.Contents, Token{TRUE, "true"})
			} else if string(buf) == "false" {
				tokens.Contents = append(tokens.Contents, Token{FALSE, "false"})
			} else {
				tokens.Contents = append(tokens.Contents, Token{IDENT, string(buf)})

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
			tokens.Contents = append(tokens.Contents, Token{NUM, string(buf)})

		case *source.peek(0) == '}' && stringInterpolationOpened:
			source.consume(1)
			stringInterpolationOpened = false
			buf := []rune{}
			for {
				if *source.peek(0) == '"' {
					break
				} else if *source.peek(0) == '{' {
					tokens.Contents = append(tokens.Contents, Token{STRING_PART, string(buf)})
					stringInterpolationOpened = true
					source.consume(1)
					break
				} else if *source.peek(0) == '\\' {
					source.consume(1)
					buf = append(buf, *source.peek(0))
					source.consume(1)
				} else {
					buf = append(buf, *source.peek(0))
					source.consume(1)
				}
			}
			if !stringInterpolationOpened {
				source.consume(1)
				tokens.Contents = append(tokens.Contents, Token{STRING_PART, string(buf)})
				tokens.Contents = append(tokens.Contents, Token{STRING_END, ""})
			}
		case *source.peek(0) == '"':
			source.consume(1)
			tokens.Contents = append(tokens.Contents, Token{STRING_START, ""})
			buf := []rune{}
			for {
				if *source.peek(0) == '"' {
					break
				} else if *source.peek(0) == '{' {
					tokens.Contents = append(tokens.Contents, Token{STRING_PART, string(buf)})
					stringInterpolationOpened = true
					source.consume(1)
					break
				} else if *source.peek(0) == '\\' {
					source.consume(1)
					buf = append(buf, *source.peek(0))
					source.consume(1)
				} else {
					buf = append(buf, *source.peek(0))
					source.consume(1)
				}
			}
			if !stringInterpolationOpened {
				source.consume(1)
				tokens.Contents = append(tokens.Contents, Token{STRING_PART, string(buf)})
				tokens.Contents = append(tokens.Contents, Token{STRING_END, ""})
			}

		case *source.peek(0) != '\n' && unicode.IsSpace(*source.peek(0)):
			source.consume(1)

		case *source.peek(0) == '\n':
			source.consume(1)
			tokens.Contents = append(tokens.Contents, Token{EOL, "\\n"})
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
					tokens.Contents = append(tokens.Contents, Token{INDENT, strconv.Itoa(currentIndentLevel)})
				} else if currentIndentLevel < indentLevel[len(indentLevel)-1] {
					tokens.Contents = append(tokens.Contents, Token{DEDENT, strconv.Itoa(indentLevel[len(indentLevel)-1])})
					indentLevel = indentLevel[:len(indentLevel)-1]
				}
			}

		case *source.peek(0) == '(':
			parenLevel += 1
			tokens.Contents = append(tokens.Contents, Token{LPAREN, strconv.Itoa(parenLevel)})
			source.consume(1)

		case *source.peek(0) == ')':
			parenLevel -= 1
			tokens.Contents = append(tokens.Contents, Token{RPAREN, strconv.Itoa(parenLevel + 1)})
			source.consume(1)

		case *source.peek(0) == '{':
			braceLevel += 1
			tokens.Contents = append(tokens.Contents, Token{LBRACE, strconv.Itoa(braceLevel)})
			source.consume(1)

		case *source.peek(0) == '}':
			braceLevel -= 1
			tokens.Contents = append(tokens.Contents, Token{RBRACE, strconv.Itoa(braceLevel + 1)})
			source.consume(1)

		case *source.peek(0) == ',':
			tokens.Contents = append(tokens.Contents, Token{COMMA, ","})
			source.consume(1)

		case *source.peek(0) == '+':
			tokens.Contents = append(tokens.Contents, Token{PLUS, "+"})
			source.consume(1)

		case *source.peek(0) == '-':
			tokens.Contents = append(tokens.Contents, Token{MINUS, "-"})
			source.consume(1)

		case *source.peek(0) == '*':
			tokens.Contents = append(tokens.Contents, Token{STAR, "*"})
			source.consume(1)

		case *source.peek(0) == '/':
			tokens.Contents = append(tokens.Contents, Token{SLASH, "/"})
			source.consume(1)

		case *source.peek(0) == ':':
			tokens.Contents = append(tokens.Contents, Token{COLON, ":"})
			source.consume(1)
		case *source.peek(0) == '<':
			tokens.Contents = append(tokens.Contents, Token{LESSER_THAN, "<"})
			source.consume(1)
		case *source.peek(0) == '>':
			tokens.Contents = append(tokens.Contents, Token{GREATER_THAN, ">"})
			source.consume(1)
		default:
			tokens.Contents = append(tokens.Contents, Token{UNKNOWN, string(*source.peek(0))})
			source.consume(1)
		}
	}
	tokens.Contents = append(tokens.Contents, Token{EOF, ""})
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

type StringLiteral struct {
	Parts []Expression
}

func (StringLiteral) Expr() {}

type StringPart struct {
	Value string
}

func (s StringPart) Expr() {}

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
	prefixParseFns[LPAREN] = parseGroupedExpression
	prefixParseFns[CASE] = parseCaseExpression
	prefixParseFns[STRING_START] = parseStringLiteral

	infixParseFns[PLUS] = parseInfixExpression
	infixParseFns[MINUS] = parseInfixExpression
	infixParseFns[STAR] = parseInfixExpression
	infixParseFns[SLASH] = parseInfixExpression
	infixParseFns[IS] = parseIsExpression
	infixParseFns[GREATER_THAN] = parseInfixExpression
	infixParseFns[LESSER_THAN] = parseInfixExpression
	infixParseFns[AND] = parseInfixExpression
	infixParseFns[OR] = parseInfixExpression
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

func parseAssignmentStatement(tokens *Stream[Token]) Statement {
	stmt := AssignmentStatement{}
	stmt.Name = parseIdentifier(tokens).(Identifier)

	tokens.consume(1)

	if !isToken(tokens, COLON, 0) {
		log.Fatal("no colon in assignment statement")
	}

	tokens.consume(1)

	stmt.Value = parseBlockExpression(tokens)

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

func parseCaseExpression(tokens *Stream[Token]) Expression {
	expr := &CaseExpression{}
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
		block := parseBlockExpression(tokens)
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

func parseBlockExpression(tokens *Stream[Token]) Block {
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
		return &BooleanLiteral{true}
	} else {
		return &BooleanLiteral{false}
	}
}

func parseInfixExpression(tokens *Stream[Token], left Expression) Expression {
	expr := &InfixExpression{Left: left, Operator: string(tokens.peek(0).Type)}
	precedence := getPrecedence(*tokens.peek(0))
	tokens.consume(1)
	expr.Right = parseExpression(tokens, precedence)
	return expr
}

func parseIsExpression(tokens *Stream[Token], left Expression) Expression {
	expr := &InfixExpression{Left: left, Operator: IS}
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
	expr := &PrefixExpression{Operator: string(tokens.peek(0).Type)}
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

func parseStringLiteral(tokens *Stream[Token]) Expression {
	tokens.consume(1)
	expr := StringLiteral{}
	for {
		if isToken(tokens, STRING_END, 0) {
			tokens.consume(1)
			break
		} else if isToken(tokens, STRING_PART, 0) {
			expr.Parts = append(expr.Parts, StringPart{tokens.peek(0).Value})
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
	log.Println(!(tokens.peek(1).Type == EOL), getPrecedence(*tokens.peek(1)))

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
