package zygonlang

import (
	"fmt"
	"strconv"

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

type Name interface {
	Expression
	Name()
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
	Name  *TableKey
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

var prefixParsers = map[TokenType]func(*Stream[Token]) Expression{}
var infixParsers = map[TokenType]func(*Stream[Token], Expression) Expression{}

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

func Parse(tokens *Stream[Token]) Program {
	program := Program{Body: []Node{}}
	prefixParsers[IDENT] = parseIdentifier
	prefixParsers[NUM] = parseNumberLiteral
	prefixParsers[NOT] = parsePrefixExpression
	prefixParsers[MINUS] = parsePrefixExpression
	prefixParsers[TRUE] = parseBooleanLiteral
	prefixParsers[FALSE] = parseBooleanLiteral
	prefixParsers[LPAREN] = resolveLParen
	prefixParsers[CASE] = parseCaseExpression
	prefixParsers[TEXT_START] = parseTextLiteral
	prefixParsers[LBRACE] = parseTableLiteral
	prefixParsers[REST] = parseRestOperator

	infixParsers[PLUS] = parseInfixExpression
	infixParsers[MINUS] = parseInfixExpression
	infixParsers[STAR] = parseInfixExpression
	infixParsers[SLASH] = parseInfixExpression
	infixParsers[IS] = parseIsExpression
	infixParsers[GREATER_THAN] = parseInfixExpression
	infixParsers[LESSER_THAN] = parseInfixExpression
	infixParsers[AND] = parseInfixExpression
	infixParsers[OR] = parseInfixExpression
	infixParsers[LPAREN] = parseFunction
	infixParsers[DOT] = parseAccessOperator
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

func parseAccessOperator(tokens *Stream[Token], left Expression) Expression {
	tokens.consume(1)
	if tokens.peek(0).Type == IDENT {
		return AccessOperator{Subject: left, Attribute: parseIdentifier(tokens)}
	} else if tokens.peek(0).Type == LPAREN {
		return AccessOperator{Subject: left, Attribute: Grouped{parseGroupedExpression(tokens)}}
	}
	panic(fmt.Sprintf("expected an IDENT or LPAREN, not %s", tokens.peek(0).Type))
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
				if isToken(tokens, COMMA, 0) {
					tokens.consume(1)
				} else if !(isToken(tokens, RPAREN, 0) && tokens.peek(0).Value == parenLevel) {
					panic("no comma in function declaration")
				}
				expr.Parameters.Set(name, param_default)

			} else if isToken(tokens, REST, 0) {
				rest := parseRestOperator(tokens).(RestOperator)
				expr.Rest = &rest
				tokens.consume(1)
			} else {
				panic(fmt.Sprintf("Expected an IDENT, not %s", tokens.peek(0).Type))
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
				argument := FunctionCallArgument{}
				if isToken(tokens, COLON, 1) {
					switch name := parseIdentifier(tokens).(type) {
					case Identifier:
						argument.Name = &TableKey{name.Value}
					}
					tokens.consume(2)
				}
				argument.Value = parseExpression(tokens, LOWEST)
				tokens.consume(1)
				if isToken(tokens, COMMA, 0) {
					tokens.consume(1)
				} else if !(isToken(tokens, RPAREN, 0) && tokens.peek(0).Value == parenLevel) {
					panic("no comma in function call")
				}
				expr.Arguments = append(expr.Arguments, argument)
			} else {
				argument := FunctionCallArgument{}
				argument.Name = nil
				argument.Value = parseExpression(tokens, LOWEST)
				tokens.consume(1)
				if isToken(tokens, COMMA, 0) {
					tokens.consume(1)
				} else if !(isToken(tokens, RPAREN, 0) && tokens.peek(0).Value == parenLevel) {
					panic("no comma in function call")
				}
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

func parseUsingPath(tokens *Stream[Token]) Name {
	if isToken(tokens, IDENT, 0) {
		if (isToken(tokens, DOT, 1) && isToken(tokens, LPAREN, 2)) || (isToken(tokens, EOL, 1) || isToken(tokens, COMMA, 1)) {
			return Identifier{tokens.peek(0).Value}
		} else if isToken(tokens, DOT, 1) {
			if isToken(tokens, IDENT, 2) {
				subject := Identifier{tokens.peek(0).Value}
				tokens.consume(2)
				return AccessOperator{Subject: subject, Attribute: parseUsingPath(tokens)}
			} else {
				panic("expected an IDENT or ( after this")
			}
		}
	}
	panic("")
}

func parseUsingStatement(tokens *Stream[Token]) Statement {
	tokens.consume(1)
	stmt := UsingStatement{}
	for {
		if isToken(tokens, EOL, 0) {
			break
		} else if isToken(tokens, IDENT, 0) {
			mod := Module{Module: parseUsingPath(tokens)}
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
						if isToken(tokens, COMMA, 0) {
							tokens.consume(1)
						} else if isToken(tokens, RPAREN, 0) {

						} else {
							panic(fmt.Sprintf("expected a COMMA or a newline, not %s", tokens.peek(0).Type))
						}
					} else if isToken(tokens, EOL, 0) {
						tokens.consume(1)
					} else if isToken(tokens, EOF, 0) {
						panic("unexpected end of .()")
					} else {
						panic(fmt.Sprintf("expected a module name, not %s", tokens.peek(0).Type))
					}
				}
				tokens.consume(1)
			}
			if isToken(tokens, COMMA, 0) {
				tokens.consume(1)
			} else if isToken(tokens, EOL, 0) {

			} else {
				panic(fmt.Sprintf("expected a COMMA or a newline, not %s", tokens.peek(0).Type))
			}
			stmt.Modules = append(stmt.Modules, mod)
		} else {
			panic(fmt.Sprintf("expected a name of a module, not %s", tokens.peek(0).Type))
		}

	}
	if len(stmt.Modules) == 0 {
		panic("expected a name of a module after this")
	}
	return stmt
}

func parseTableLiteral(tokens *Stream[Token]) Expression {
	expr := TableLiteral{}
	braceLevel := tokens.peek(0).Value
	tokens.consume(1)
	for !(isToken(tokens, RBRACE, 0) && tokens.peek(0).Value == braceLevel) {
		entry := TableEntry{}
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
			expr.Cases = append(expr.Cases, CaseExpressionCase{pattern, block})
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
		panic("expected INDENT or a value after this")
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
	prefix := prefixParsers[tokens.peek(0).Type]
	if prefix == nil {
		panic(fmt.Sprintf("Did not expect %s", tokens.peek(0).Type))
	}
	leftExpr := prefix(tokens)

	for !(tokens.peek(1).Type == EOL) && precedence < getPrecedence(*tokens.peek(1)) {
		infix := infixParsers[tokens.peek(1).Type]
		if infix == nil {
			return leftExpr
		}
		tokens.consume(1)
		leftExpr = infix(tokens, leftExpr)
	}

	return leftExpr
}
