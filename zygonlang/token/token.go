package token

import (
	"strconv"
	"thechosenzendro/zygonlang/zygonlang/stream"
	"unicode"
)

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

func Tokenize(sourceCode string) stream.Stream[Token] {
	source := &stream.Stream[rune]{Index: 0, Contents: []rune(sourceCode)}
	tokens := &stream.Stream[Token]{Index: 0, Contents: []Token{}}
	for source.Peek(0) != nil {
		tokens.Contents = append(tokens.Contents, lexToken(source)...)
	}
	tokens.Contents = append(tokens.Contents, Token{EOF, ""})
	return *tokens
}

func IsToken(tokens *stream.Stream[Token], tokenType TokenType, amount int) bool {
	index := tokens.Index + amount

	if index >= len(tokens.Contents) {
		return false
	}
	return tokens.Contents[index].Type == tokenType
}
func lexToken(source *stream.Stream[rune]) []Token {
	tokens := []Token{}

	switch {

	case *source.Peek(0) == '#':
		for *source.Peek(0) != '\n' {
			source.Consume(1)
		}
	case unicode.IsLetter(*source.Peek(0)) || *source.Peek(0) == '_':
		buf := []rune{}
		for source.Peek(0) != nil && (unicode.IsLetter(*source.Peek(0)) || *source.Peek(0) == '_' || unicode.IsDigit(*source.Peek(0))) {
			buf = append(buf, *source.Peek(0))
			source.Consume(1)
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
	case *source.Peek(0) == '.':
		if *source.Peek(1) == '.' && *source.Peek(2) == '.' {
			tokens = append(tokens, Token{REST, "..."})
			source.Consume(3)
		} else {
			tokens = append(tokens, Token{DOT, "."})
			source.Consume(1)
		}
	case unicode.IsDigit(*source.Peek(0)):
		buf := []rune{}
		hasDecimal := false
		for source.Peek(0) != nil && (unicode.IsDigit(*source.Peek(0)) || *source.Peek(0) == '_' || *source.Peek(0) == '.') {
			if *source.Peek(0) == '.' {
				if hasDecimal {
					panic("Number literal cannot have more decimal parts")
				} else {
					hasDecimal = true
				}
			}
			if *source.Peek(0) == '_' {
				source.Consume(1)
			} else {
				buf = append(buf, *source.Peek(0))
				source.Consume(1)
			}
		}
		if buf[len(buf)-1] == '.' {
			panic("Expected fractional part after DOT in number literal")
		}
		tokens = append(tokens, Token{NUM, string(buf)})

	case *source.Peek(0) == '"':
		source.Consume(1)
		tokens = append(tokens, Token{TEXT_START, ""})
		buf := []rune{}
		for {
			if source.Peek(0) == nil {
				panic("Unterminated text literal")
			}
			if *source.Peek(0) == '"' {
				break
			} else if *source.Peek(0) == '{' {
				braceLevel += 1
				bl := braceLevel
				tokens = append(tokens, Token{TEXT_PART, string(buf)})
				buf = []rune{}
				source.Consume(1)
				for *source.Peek(0) != '}' && braceLevel == bl {
					i := 0
					for {
						if source.Peek(i) == nil {
							panic("Unterminated interpolation in text literal")
						}
						if *source.Peek(i) == '}' && braceLevel == bl {
							break
						}
						i += 1
					}
					tokens = append(tokens, lexToken(source)...)
				}
				source.Consume(1)
				braceLevel -= 1

			} else if *source.Peek(0) == '\\' {
				source.Consume(1)
				if *source.Peek(0) == 'n' {
					buf = append(buf, '\n')
				} else if *source.Peek(0) == 't' {
					buf = append(buf, '\t')
				} else {
					buf = append(buf, *source.Peek(0))
				}
				source.Consume(1)
			} else {
				buf = append(buf, *source.Peek(0))
				source.Consume(1)
			}
		}
		source.Consume(1)
		tokens = append(tokens, Token{TEXT_PART, string(buf)})
		tokens = append(tokens, Token{TEXT_END, ""})

	case *source.Peek(0) != '\n' && unicode.IsSpace(*source.Peek(0)):
		source.Consume(1)

	case *source.Peek(0) == '\n':
		source.Consume(1)
		if parenLevel == 0 {
			tokens = append(tokens, Token{EOL, "\\n"})
			currentIndentLevel := 0

			if source.Peek(0) != nil {
				for {
					if *source.Peek(0) == ' ' {
						currentIndentLevel += 1
						source.Consume(1)

					} else if *source.Peek(0) == '\t' {
						currentIndentLevel += 4
						source.Consume(1)

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

		}

	case *source.Peek(0) == '(':
		parenLevel += 1
		tokens = append(tokens, Token{LPAREN, strconv.Itoa(parenLevel)})
		source.Consume(1)

	case *source.Peek(0) == ')':
		parenLevel -= 1
		tokens = append(tokens, Token{RPAREN, strconv.Itoa(parenLevel + 1)})
		source.Consume(1)

	case *source.Peek(0) == '{':
		braceLevel += 1
		tokens = append(tokens, Token{LBRACE, strconv.Itoa(braceLevel)})
		source.Consume(1)

	case *source.Peek(0) == '}':
		braceLevel -= 1
		tokens = append(tokens, Token{RBRACE, strconv.Itoa(braceLevel + 1)})
		source.Consume(1)

	case *source.Peek(0) == ',':
		tokens = append(tokens, Token{COMMA, ","})
		source.Consume(1)

	case *source.Peek(0) == '+':
		tokens = append(tokens, Token{PLUS, "+"})
		source.Consume(1)

	case *source.Peek(0) == '-':
		tokens = append(tokens, Token{MINUS, "-"})
		source.Consume(1)

	case *source.Peek(0) == '*':
		tokens = append(tokens, Token{STAR, "*"})
		source.Consume(1)

	case *source.Peek(0) == '/':
		tokens = append(tokens, Token{SLASH, "/"})
		source.Consume(1)

	case *source.Peek(0) == ':':
		tokens = append(tokens, Token{COLON, ":"})
		source.Consume(1)
	case *source.Peek(0) == '<':
		tokens = append(tokens, Token{LESSER_THAN, "<"})
		source.Consume(1)
	case *source.Peek(0) == '>':
		tokens = append(tokens, Token{GREATER_THAN, ">"})
		source.Consume(1)
	default:
		tokens = append(tokens, Token{UNKNOWN, string(*source.Peek(0))})
		source.Consume(1)
	}
	return tokens
}
