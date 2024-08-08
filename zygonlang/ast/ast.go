package ast

import (
	"fmt"
	"strconv"
	"thechosenzendro/zygonlang/zygonlang/stream"
	"thechosenzendro/zygonlang/zygonlang/token"

	"github.com/elliotchance/orderedmap/v2"
)

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
	Value string
}

func (Identifier) Expr() {}
func (Identifier) Name() {}

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

func (TextPart) Expr() {}

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
	Cases   []CaseExpressionCase
	Default *Block
}

type CaseExpressionCase struct {
	Pattern Expression
	Block   Block
}

func (CaseExpression) Expr() {}

type Block struct {
	Body []Node
}

type UsingStatement struct {
	Modules []Module
}

type Module struct {
	Module  Name
	Symbols []Identifier
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

type FunctionCallArgument struct {
	Name  *Identifier
	Value Expression
}

type FunctionCall struct {
	Fn        Expression
	Arguments []FunctionCallArgument
}

func (FunctionCall) Expr() {}

type TableEntry struct {
	Key   *Identifier
	Value Expression
}
type TableLiteral struct {
	Entries []TableEntry
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

func (AccessOperator) Expr() {}
func (AccessOperator) Name() {}

type RestOperator struct {
	Value Expression
}

func (RestOperator) Expr() {}

type Name interface {
	Expression
	Name()
}

var prefixParsers = map[token.TokenType]func(*stream.Stream[token.Token]) Expression{}
var infixParsers = map[token.TokenType]func(*stream.Stream[token.Token], Expression) Expression{}

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

var precedences = map[token.TokenType]int{
	token.DOT:          ACCESS,
	token.IS:           EQUALS,
	token.LESSER_THAN:  LESSGREATER,
	token.GREATER_THAN: LESSGREATER,
	token.PLUS:         SUM,
	token.MINUS:        SUM,
	token.STAR:         PRODUCT,
	token.SLASH:        PRODUCT,
	token.AND:          ANDPREC,
	token.OR:           ORPREC,
	token.LPAREN:       CALL,
}

func getPrecedence(token token.Token) int {
	if precedence, ok := precedences[token.Type]; ok {
		return precedence
	}
	return LOWEST
}

func Parse(tokens *stream.Stream[token.Token]) Program {
	program := Program{Body: []Node{}}
	prefixParsers[token.IDENT] = parseIdentifier
	prefixParsers[token.NUM] = parseNumberLiteral
	prefixParsers[token.NOT] = parsePrefixExpression
	prefixParsers[token.MINUS] = parsePrefixExpression
	prefixParsers[token.TRUE] = parseBooleanLiteral
	prefixParsers[token.FALSE] = parseBooleanLiteral
	prefixParsers[token.LPAREN] = resolveLParen
	prefixParsers[token.CASE] = parseCaseExpression
	prefixParsers[token.TEXT_START] = parseTextLiteral
	prefixParsers[token.LBRACE] = parseTableLiteral
	prefixParsers[token.REST] = parseRestOperator

	infixParsers[token.PLUS] = parseInfixExpression
	infixParsers[token.MINUS] = parseInfixExpression
	infixParsers[token.STAR] = parseInfixExpression
	infixParsers[token.SLASH] = parseInfixExpression
	infixParsers[token.IS] = parseIsExpression
	infixParsers[token.GREATER_THAN] = parseInfixExpression
	infixParsers[token.LESSER_THAN] = parseInfixExpression
	infixParsers[token.AND] = parseInfixExpression
	infixParsers[token.OR] = parseInfixExpression
	infixParsers[token.LPAREN] = parseFunction
	infixParsers[token.DOT] = parseAccessOperator
	for tokens.Peek(0).Type != token.EOF {
		if tokens.Peek(0).Type != token.EOL {
			var node Node = nil
			if token.IsToken(tokens, token.IDENT, 0) && token.IsToken(tokens, token.COLON, 1) {
				node = parseAssignmentStatement(tokens)
			} else if token.IsToken(tokens, token.USING, 0) {
				node = parseUsingStatement(tokens)
			} else if token.IsToken(tokens, token.PUB, 0) {
				node = parsePubStatement(tokens)
			} else {
				node = parseExpression(tokens, LOWEST)
			}
			program.Body = append(program.Body, node)
		}
		tokens.Consume(1)
	}
	return program
}

func parseGroupedExpression(tokens *stream.Stream[token.Token]) Expression {
	tokens.Consume(1)
	expr := parseExpression(tokens, LOWEST)
	if tokens.Peek(1).Type != token.RPAREN {
		panic("no rparen")
	}
	tokens.Consume(1)
	return expr
}

func parseRestOperator(tokens *stream.Stream[token.Token]) Expression {
	expr := RestOperator{}
	if token.IsToken(tokens, token.RBRACE, 1) || token.IsToken(tokens, token.RPAREN, 1) || token.IsToken(tokens, token.EOL, 1) {
		return expr
	}
	tokens.Consume(1)
	expr.Value = parseExpression(tokens, LOWEST)
	// ...tokens, ...{}, ...(get(x))
	return expr
}

func parseAccessOperator(tokens *stream.Stream[token.Token], left Expression) Expression {
	tokens.Consume(1)
	if tokens.Peek(0).Type == token.IDENT {
		return AccessOperator{Subject: left, Attribute: parseIdentifier(tokens)}
	} else if tokens.Peek(0).Type == token.LPAREN {
		return AccessOperator{Subject: left, Attribute: Grouped{parseGroupedExpression(tokens)}}
	}
	panic(fmt.Sprintf("expected an IDENT or LPAREN, not %s", tokens.Peek(0).Type))
}

func resolveLParen(tokens *stream.Stream[token.Token]) Expression {
	i := 0
	parenLevel := tokens.Peek(0).Value

	for !(token.IsToken(tokens, token.RPAREN, i) && tokens.Peek(i).Value == parenLevel) {
		i += 1
	}
	i += 1
	if token.IsToken(tokens, token.COLON, i) {
		return parseFunction(tokens, nil)
	} else {
		return parseGroupedExpression(tokens)
	}
}

var parsingCase = false

func parseFunction(tokens *stream.Stream[token.Token], fn Expression) Expression {
	isDeclaration := false
	if fn == nil {
		isDeclaration = true
	} else {
		i := 0
		parenLevel := tokens.Peek(0).Value

		for !(token.IsToken(tokens, token.RPAREN, i) && tokens.Peek(i).Value == parenLevel) {
			i += 1
		}
		i += 1
		if token.IsToken(tokens, token.COLON, i) && !parsingCase {
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
		parenLevel := tokens.Peek(0).Value
		tokens.Consume(1)
		for !(token.IsToken(tokens, token.RPAREN, 0) && tokens.Peek(0).Value == parenLevel) {
			if token.IsToken(tokens, token.IDENT, 0) {
				name := parseIdentifier(tokens).(Identifier)
				var param_default Expression = nil
				tokens.Consume(1)
				if token.IsToken(tokens, token.COLON, 0) {
					tokens.Consume(1)
					param_default = parseExpression(tokens, LOWEST)
					tokens.Consume(1)
				}
				if token.IsToken(tokens, token.COMMA, 0) {
					tokens.Consume(1)
				} else if !(token.IsToken(tokens, token.RPAREN, 0) && tokens.Peek(0).Value == parenLevel) {
					panic("no comma in function declaration")
				}
				expr.Parameters.Set(name, param_default)

			} else if token.IsToken(tokens, token.REST, 0) {
				rest := parseRestOperator(tokens).(RestOperator)
				expr.Rest = &rest
				tokens.Consume(1)
			} else {
				panic(fmt.Sprintf("Expected an IDENT, not %s", tokens.Peek(0).Type))
			}
		}
		tokens.Consume(1)
		if !token.IsToken(tokens, token.COLON, 0) {
			panic("no colon in function declaration")
		}
		tokens.Consume(1)
		expr.Body = parseBlock(tokens)
		return expr
	} else {
		expr := FunctionCall{}
		expr.Fn = fn

		parenLevel := tokens.Peek(0).Value
		tokens.Consume(1)
		for !(token.IsToken(tokens, token.RPAREN, 0) && tokens.Peek(0).Value == parenLevel) {
			if token.IsToken(tokens, token.IDENT, 0) {
				argument := FunctionCallArgument{}
				if token.IsToken(tokens, token.COLON, 1) {
					switch name := parseIdentifier(tokens).(type) {
					case Identifier:
						argument.Name = &Identifier{name.Value}
					}
					tokens.Consume(2)
				}
				argument.Value = parseExpression(tokens, LOWEST)
				tokens.Consume(1)
				if token.IsToken(tokens, token.COMMA, 0) {
					tokens.Consume(1)
				} else if !(token.IsToken(tokens, token.RPAREN, 0) && tokens.Peek(0).Value == parenLevel) {
					panic("no comma in function call")
				}
				expr.Arguments = append(expr.Arguments, argument)
			} else {
				argument := FunctionCallArgument{}
				argument.Name = nil
				argument.Value = parseExpression(tokens, LOWEST)
				tokens.Consume(1)
				if token.IsToken(tokens, token.COMMA, 0) {
					tokens.Consume(1)
				} else if !(token.IsToken(tokens, token.RPAREN, 0) && tokens.Peek(0).Value == parenLevel) {
					panic("no comma in function call")
				}
				expr.Arguments = append(expr.Arguments, argument)
			}
		}
		return expr
	}
}
func parseAssignmentStatement(tokens *stream.Stream[token.Token]) Statement {
	stmt := AssignmentStatement{}
	stmt.Name = parseIdentifier(tokens).(Identifier)

	tokens.Consume(1)

	if !token.IsToken(tokens, token.COLON, 0) {
		panic("no colon in assignment statement")
	}

	tokens.Consume(1)

	stmt.Value = parseBlock(tokens)

	return stmt
}

func parsePubStatement(tokens *stream.Stream[token.Token]) Statement {
	stmt := PubStatement{}
	tokens.Consume(1)
	if token.IsToken(tokens, token.IDENT, 0) && token.IsToken(tokens, token.COLON, 1) {
		stmt.Public = parseAssignmentStatement(tokens)
	} else {
		stmt.Public = parseExpression(tokens, LOWEST)
	}
	return stmt
}

func parseUsingPath(tokens *stream.Stream[token.Token]) Name {
	if token.IsToken(tokens, token.IDENT, 0) {
		if (token.IsToken(tokens, token.DOT, 1) && token.IsToken(tokens, token.LPAREN, 2)) || (token.IsToken(tokens, token.EOL, 1) || token.IsToken(tokens, token.COMMA, 1)) {
			return Identifier{tokens.Peek(0).Value}
		} else if token.IsToken(tokens, token.DOT, 1) {
			if token.IsToken(tokens, token.IDENT, 2) {
				subject := Identifier{tokens.Peek(0).Value}
				tokens.Consume(2)
				return AccessOperator{Subject: subject, Attribute: parseUsingPath(tokens)}
			} else {
				panic("expected an IDENT or ( after this")
			}
		}
	}
	panic("")
}

func parseUsingStatement(tokens *stream.Stream[token.Token]) Statement {
	tokens.Consume(1)
	stmt := UsingStatement{}
	for {
		if token.IsToken(tokens, token.EOL, 0) {
			break
		} else if token.IsToken(tokens, token.IDENT, 0) {
			mod := Module{Module: parseUsingPath(tokens)}
			tokens.Consume(1)
			endsWithDot := false
			if token.IsToken(tokens, token.DOT, 0) {
				endsWithDot = true
			}

			if endsWithDot {
				tokens.Consume(1)
				if !token.IsToken(tokens, token.LPAREN, 0) {
					panic("no left paren in using")
				}

				tokens.Consume(1)
				for {
					if token.IsToken(tokens, token.RPAREN, 0) {
						break
					} else if token.IsToken(tokens, token.IDENT, 0) {
						mod.Symbols = append(mod.Symbols, parseIdentifier(tokens).(Identifier))
						tokens.Consume(1)
						if token.IsToken(tokens, token.COMMA, 0) {
							tokens.Consume(1)
						} else if token.IsToken(tokens, token.RPAREN, 0) {

						} else {
							panic(fmt.Sprintf("expected a COMMA or a newline, not %s", tokens.Peek(0).Type))
						}
					} else if token.IsToken(tokens, token.EOL, 0) {
						tokens.Consume(1)
					} else if token.IsToken(tokens, token.EOF, 0) {
						panic("unexpected end of .()")
					} else {
						panic(fmt.Sprintf("expected a module name, not %s", tokens.Peek(0).Type))
					}
				}
				tokens.Consume(1)
			}
			if token.IsToken(tokens, token.COMMA, 0) {
				tokens.Consume(1)
			} else if token.IsToken(tokens, token.EOL, 0) {

			} else {
				panic(fmt.Sprintf("expected a COMMA or a newline, not %s", tokens.Peek(0).Type))
			}
			stmt.Modules = append(stmt.Modules, mod)
		} else {
			panic(fmt.Sprintf("expected a name of a module, not %s", tokens.Peek(0).Type))
		}

	}
	if len(stmt.Modules) == 0 {
		panic("expected a name of a module after this")
	}
	return stmt
}

func parseTableLiteral(tokens *stream.Stream[token.Token]) Expression {
	expr := TableLiteral{}
	braceLevel := tokens.Peek(0).Value
	tokens.Consume(1)
	for !(token.IsToken(tokens, token.RBRACE, 0) && tokens.Peek(0).Value == braceLevel) {
		entry := TableEntry{}
		if token.IsToken(tokens, token.IDENT, 0) {
			val := parseIdentifier(tokens)

			tokens.Consume(1)

			if token.IsToken(tokens, token.COLON, 0) {
				switch key := val.(type) {
				case Identifier:
					entry.Key = &key
				}
				tokens.Consume(1)
				entry.Value = parseExpression(tokens, LOWEST)
				tokens.Consume(1)
				expr.Entries = append(expr.Entries, entry)
			} else {
				entry.Key = nil
				entry.Value = val
				expr.Entries = append(expr.Entries, entry)
			}
		} else if token.IsToken(tokens, token.COMMA, 0) {
			tokens.Consume(1)
		} else if token.IsToken(tokens, token.EOL, 0) || token.IsToken(tokens, token.INDENT, 0) || token.IsToken(tokens, token.DEDENT, 0) {
			tokens.Consume(1)
		} else {
			entry.Key = nil
			entry.Value = parseExpression(tokens, LOWEST)
			tokens.Consume(1)

			expr.Entries = append(expr.Entries, entry)
		}
	}
	return expr
}

func parseCaseExpression(tokens *stream.Stream[token.Token]) Expression {
	expr := CaseExpression{}
	parsingCase = true
	tokens.Consume(1)
	if !token.IsToken(tokens, token.COLON, 0) {
		expr.Subject = parseExpression(tokens, LOWEST)
		tokens.Consume(1)
	}
	if !token.IsToken(tokens, token.COLON, 0) {
		panic("no colon in case expression")
	}
	tokens.Consume(1)
	if !token.IsToken(tokens, token.EOL, 0) {
		panic("no newline in case expression")
	}
	tokens.Consume(1)

	if !token.IsToken(tokens, token.INDENT, 0) {
		panic("case expression must contain indentation")
	}
	indentLevel := tokens.Peek(0).Value
	tokens.Consume(1)
	for {
		tok := tokens.Peek(0)
		if tok.Type == token.DEDENT && tok.Value == indentLevel {
			break
		}
		if token.IsToken(tokens, token.DEFAULT, 0) {
			tokens.Consume(1)
			if !token.IsToken(tokens, token.COLON, 0) {
				panic("no colon in case expression")
			}
			tokens.Consume(1)
			block := parseBlock(tokens)
			tokens.Consume(1)
			if token.IsToken(tokens, token.EOL, 0) {
				tokens.Consume(1)
			}
			if expr.Default == nil {
				expr.Default = &block
			} else {
				panic("cannot have more than one default in case")
			}
		} else {
			pattern := parseExpression(tokens, LOWEST)
			tokens.Consume(1)
			if !token.IsToken(tokens, token.COLON, 0) {
				panic("no colon in case expression")
			}
			tokens.Consume(1)
			block := parseBlock(tokens)
			tokens.Consume(1)
			if token.IsToken(tokens, token.EOL, 0) {
				tokens.Consume(1)
			}
			expr.Cases = append(expr.Cases, CaseExpressionCase{pattern, block})
		}
	}
	parsingCase = false
	return expr
}

func parseBlock(tokens *stream.Stream[token.Token]) Block {
	block := Block{}
	if !token.IsToken(tokens, token.EOL, 0) {
		expr := parseExpression(tokens, LOWEST)
		block.Body = []Node{expr}
		return block
	}
	tokens.Consume(1)
	if !token.IsToken(tokens, token.INDENT, 0) {
		panic("expected INDENT or a value after this")
	}
	indentLevel := tokens.Peek(0).Value
	tokens.Consume(1)
	for {
		tok := tokens.Peek(0)
		if tok.Type == token.EOL {
			tokens.Consume(1)
		}
		tok = tokens.Peek(0)

		if tok.Type == token.DEDENT && tok.Value == indentLevel {
			break
		}
		var node Node = nil
		if token.IsToken(tokens, token.IDENT, 0) && token.IsToken(tokens, token.COLON, 1) {
			node = parseAssignmentStatement(tokens)
		} else if token.IsToken(tokens, token.USING, 0) {
			node = parseUsingStatement(tokens)
		} else if token.IsToken(tokens, token.PUB, 0) {
			node = parsePubStatement(tokens)
		} else {
			node = parseExpression(tokens, LOWEST)
		}
		block.Body = append(block.Body, node)
		tokens.Consume(1)
	}
	return block
}

func parseBooleanLiteral(tokens *stream.Stream[token.Token]) Expression {
	if token.IsToken(tokens, token.TRUE, 0) {
		return BooleanLiteral{true}
	} else {
		return BooleanLiteral{false}
	}
}

func parseInfixExpression(tokens *stream.Stream[token.Token], left Expression) Expression {
	expr := InfixExpression{Left: left, Operator: string(tokens.Peek(0).Type)}
	precedence := getPrecedence(*tokens.Peek(0))
	tokens.Consume(1)
	expr.Right = parseExpression(tokens, precedence)
	return expr
}

func parseIsExpression(tokens *stream.Stream[token.Token], left Expression) Expression {
	expr := InfixExpression{Left: left, Operator: token.IS}
	precedence := getPrecedence(*tokens.Peek(0))
	tokens.Consume(1)
	if token.IsToken(tokens, token.NOT, 0) {
		expr.Operator = token.IS_NOT
		tokens.Consume(1)
	}
	expr.Right = parseExpression(tokens, precedence)
	return expr
}

func parsePrefixExpression(tokens *stream.Stream[token.Token]) Expression {
	expr := PrefixExpression{Operator: string(tokens.Peek(0).Type)}
	tokens.Consume(1)
	expr.Right = parseExpression(tokens, PREFIX)
	return expr
}

func parseNumberLiteral(tokens *stream.Stream[token.Token]) Expression {
	num, err := strconv.ParseFloat(tokens.Peek(0).Value, 64)
	if err != nil {
		panic(err)
	}
	return NumberLiteral{num}
}

func parseIdentifier(tokens *stream.Stream[token.Token]) Expression {
	return Identifier{tokens.Peek(0).Value}
}

func parseTextLiteral(tokens *stream.Stream[token.Token]) Expression {
	tokens.Consume(1)
	expr := TextLiteral{}
	for {
		if token.IsToken(tokens, token.TEXT_END, 0) {
			break
		} else if token.IsToken(tokens, token.TEXT_PART, 0) {
			expr.Parts = append(expr.Parts, TextPart{tokens.Peek(0).Value})
			tokens.Consume(1)
		} else {
			expr.Parts = append(expr.Parts, parseExpression(tokens, LOWEST))
			tokens.Consume(1)
		}
	}
	return expr
}

func parseExpression(tokens *stream.Stream[token.Token], precedence int) Expression {
	prefix := prefixParsers[tokens.Peek(0).Type]
	if prefix == nil {
		panic(fmt.Sprintf("Did not expect %s", tokens.Peek(0).Type))
	}
	leftExpr := prefix(tokens)

	for !(tokens.Peek(1).Type == token.EOL) && precedence < getPrecedence(*tokens.Peek(1)) {
		infix := infixParsers[tokens.Peek(1).Type]
		if infix == nil {
			return leftExpr
		}
		tokens.Consume(1)
		leftExpr = infix(tokens, leftExpr)
	}

	return leftExpr
}
