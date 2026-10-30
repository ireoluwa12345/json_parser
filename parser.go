package main

import (
	"encoding/json"
	"fmt"
)

type Parser struct {
	lexer   *Lexer
	current Token
	loaded  bool
}

func NewParser(lexer *Lexer) *Parser {
	return &Parser{lexer: lexer}
}

func (p *Parser) Parse() (any, error) {
	value, err := p.parseValue()
	if err != nil {
		return nil, err
	}
	tok, err := p.next()
	if err != nil {
		return nil, err
	}
	if tok.Type != TokenEOF {
		return nil, p.unexpected(tok, "end of input")
	}
	return value, nil
}

func (p *Parser) peek() (Token, error) {
	if !p.loaded {
		tok, err := p.lexer.NextToken()
		if err != nil {
			return Token{}, err
		}
		p.current = tok
		p.loaded = true
	}
	return p.current, nil
}

func (p *Parser) next() (Token, error) {
	tok, err := p.peek()
	if err != nil {
		return Token{}, err
	}
	p.loaded = false
	return tok, nil
}

func (p *Parser) parseValue() (any, error) {
	tok, err := p.next()
	if err != nil {
		return nil, err
	}
	switch tok.Type {
	case TokenString:
		return tok.Value, nil
	case TokenNumber:
		return json.Number(tok.Value), nil
	case TokenTrue:
		return true, nil
	case TokenFalse:
		return false, nil
	case TokenNull:
		return nil, nil
	case TokenLeftBracket:
		return p.parseArray()
	case TokenLeftBrace:
		return p.parseArray()
	default:
		return nil, p.unexpected(tok, "JSON value")
	}
}

func (p *Parser) parseArray() ([]any, error) {
	values := make([]any, 0)
	tok, err := p.peek()
	if err != nil {
		return nil, err
	}
	if tok.Type == TokenRightBracket {
		_, _ = p.next()
		return values, nil
	}

	for {
		value, err := p.parseValue()
		if err != nil {
			return nil, err
		}
		values = append(values, value)

		tok, err = p.next()
		if err != nil {
			return nil, err
		}
		switch tok.Type {
		case TokenRightBracket:
			return values, nil
		case TokenComma:
		default:
			return nil, p.unexpected(tok, "',' or ']'")
		}
	}
}

func (p *Parser) parseObject() (map[string]any, error) {
	values := make(map[string]any)
	tok, err := p.peek()
	if err != nil {
		return nil, err
	}
	if tok.Type == TokenRightBrace {
		_, _ = p.next()
		return values, nil
	}

	for {
		keyToken, err := p.next()
		if err != nil {
			return nil, err
		}
		if keyToken.Type != TokenString {
			return nil, p.unexpected(keyToken, "string object key")
		}

		colon, err := p.next()
		if err != nil {
			return nil, err
		}
		if colon.Type != TokenColon {
			return nil, p.unexpected(colon, "':'")
		}

		value, err := p.parseValue()
		if err != nil {
			return nil, err
		}
		values[keyToken.Value] = value

		sep, err := p.next()
		if err != nil {
			return nil, err
		}
		switch sep.Type {
		case TokenRightBrace:
			return values, nil
		case TokenComma:
			// The next loop requires a string key, rejecting a trailing comma.
		default:
			return nil, p.unexpected(sep, "',' or '}'")
		}
	}
}

func (p *Parser) unexpected(tok Token, expected string) error {
	return fmt.Errorf("unexpected %s at line %d, column %d: expected %s", tok.Type, tok.Position.Line, tok.Position.Column, expected)
}
