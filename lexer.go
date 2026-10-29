package main

import (
	"encoding/json"
	"fmt"
	"unicode/utf8"
)

type TokenType uint8

// JSON Tokens = /n, {, }, [, ], :, ,, string, number, true, false, null

const (
	TokenIllegal TokenType = iota
	TokenEOF
	TokenLeftBrace
	TokenRightBrace
	TokenLeftBracket
	TokenRightBracket
	TokenColon
	TokenComma
	TokenString
	TokenNumber
	TokenTrue
	TokenFalse
	TokenNull
)

func (t TokenType) String() string {
	switch t {
	case TokenEOF:
		return "end of input"
	case TokenLeftBrace:
		return "{"
	case TokenRightBrace:
		return "}"
	case TokenLeftBracket:
		return "["
	case TokenRightBracket:
		return "]"
	case TokenColon:
		return ":"
	case TokenComma:
		return ","
	case TokenString:
		return "string"
	case TokenNumber:
		return "number"
	case TokenTrue:
		return "true"
	case TokenFalse:
		return "false"
	case TokenNull:
		return "null"
	default:
		return "illegal token"
	}
}

type Position struct {
	Offset int
	Line   int
	Column int
}

type Token struct {
	Type     TokenType
	Value    string
	Position Position
}

type Lexer struct {
	input  string
	offset int
	line   int
	column int
}

func NewLexer(input []byte) *Lexer {
	return &Lexer{input: string(input), line: 1, column: 1}
}

func (l *Lexer) position() Position {
	return Position{Offset: l.offset, Line: l.line, Column: l.column}
}

func (l *Lexer) peek() byte {
	if l.offset >= len(l.input) {
		return 0
	}
	return l.input[l.offset]
}

func (l *Lexer) advance() byte {
	if l.offset >= len(l.input) {
		return 0
	}
	b := l.input[l.offset]
	l.offset++
	if b == '\n' {
		l.line++
		l.column = 1
	} else {
		l.column++
	}
	return b
}

func (l *Lexer) skipWhitespace() {
	for {
		switch l.peek() {
		case ' ', '\t', '\r', '\n':
			l.advance()
		default:
			return
		}
	}
}

func (l *Lexer) NextToken() (Token, error) {
	l.skipWhitespace()
	pos := l.position()
	if l.offset == len(l.input) {
		return Token{Type: TokenEOF, Position: pos}, nil
	}

	switch b := l.peek(); b {
	case '{':
		l.advance()
		return Token{Type: TokenLeftBrace, Value: "{", Position: pos}, nil
	case '}':
		l.advance()
		return Token{Type: TokenRightBrace, Value: "}", Position: pos}, nil
	case '[':
		l.advance()
		return Token{Type: TokenLeftBracket, Value: "[", Position: pos}, nil
	case ']':
		l.advance()
		return Token{Type: TokenRightBracket, Value: "]", Position: pos}, nil
	case ':':
		l.advance()
		return Token{Type: TokenColon, Value: ":", Position: pos}, nil
	case ',':
		l.advance()
		return Token{Type: TokenComma, Value: ",", Position: pos}, nil
	case '"':
		return l.scanString(pos)
	case 't':
		return l.scanKeyword("true", TokenTrue, pos)
	case 'f':
		return l.scanKeyword("false", TokenFalse, pos)
	case 'n':
		return l.scanKeyword("null", TokenNull, pos)
	default:
		if b == '-' || (b >= '0' && b <= '9') {
			return l.scanNumber(pos)
		}
		if b >= utf8.RuneSelf {
			return Token{}, fmt.Errorf("invalid non-ASCII character at line %d, column %d", pos.Line, pos.Column)
		}
		return Token{}, fmt.Errorf("unexpected character %q at line %d, column %d", b, pos.Line, pos.Column)
	}
}

func (l *Lexer) scanKeyword(word string, typ TokenType, pos Position) (Token, error) {
	start := l.offset
	for i := 0; i < len(word); i++ {
		if l.peek() != word[i] {
			return Token{}, fmt.Errorf("invalid token at line %d, column %d: expected %q", pos.Line, pos.Column, word)
		}
		l.advance()
	}
	return Token{Type: typ, Value: l.input[start:l.offset], Position: pos}, nil
}

func (l *Lexer) scanString(pos Position) (Token, error) {
	start := l.offset
	l.advance() // opening quote
	for l.offset < len(l.input) {
		b := l.peek()
		switch b {
		case '"':
			l.advance()
			raw := l.input[start:l.offset]
			if !utf8.ValidString(raw) {
				return Token{}, fmt.Errorf("invalid UTF-8 in string at line %d, column %d", pos.Line, pos.Column)
			}
			var value string
			if err := json.Unmarshal([]byte(raw), &value); err != nil {
				return Token{}, fmt.Errorf("invalid string at line %d, column %d: %w", pos.Line, pos.Column, err)
			}
			return Token{Type: TokenString, Value: value, Position: pos}, nil
		case '\\':
			l.advance()
			if l.offset == len(l.input) {
				return Token{}, fmt.Errorf("unterminated escape in string at line %d, column %d", pos.Line, pos.Column)
			}
			l.advance()
		case '\n', '\r':
			return Token{}, fmt.Errorf("unescaped newline in string at line %d, column %d", l.line, l.column)
		default:
			if b < 0x20 {
				return Token{}, fmt.Errorf("unescaped control character in string at line %d, column %d", l.line, l.column)
			}
			l.advance()
		}
	}
	return Token{}, fmt.Errorf("unterminated string at line %d, column %d", pos.Line, pos.Column)
}

func (l *Lexer) scanNumber(pos Position) (Token, error) {
	start := l.offset
	if l.peek() == '-' {
		l.advance()
	}

	switch b := l.peek(); {
	case b == '0':
		l.advance()
		if digit(l.peek()) {
			return Token{}, fmt.Errorf("leading zero in number at line %d, column %d", pos.Line, pos.Column)
		}
	case b >= '1' && b <= '9':
		for digit(l.peek()) {
			l.advance()
		}
	default:
		return Token{}, fmt.Errorf("invalid number at line %d, column %d", pos.Line, pos.Column)
	}

	if l.peek() == '.' {
		l.advance()
		if !digit(l.peek()) {
			return Token{}, fmt.Errorf("expected digit after decimal point at line %d, column %d", l.line, l.column)
		}
		for digit(l.peek()) {
			l.advance()
		}
	}

	if l.peek() == 'e' || l.peek() == 'E' {
		l.advance()
		if l.peek() == '+' || l.peek() == '-' {
			l.advance()
		}
		if !digit(l.peek()) {
			return Token{}, fmt.Errorf("expected digit in exponent at line %d, column %d", l.line, l.column)
		}
		for digit(l.peek()) {
			l.advance()
		}
	}

	return Token{Type: TokenNumber, Value: l.input[start:l.offset], Position: pos}, nil
}

func digit(b byte) bool { return b >= '0' && b <= '9' }
