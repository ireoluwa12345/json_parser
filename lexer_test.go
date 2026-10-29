package main

import (
	"testing"
)

// lexAll runs the lexer to completion and returns every token up to and
// including EOF, or the first error encountered.
func lexAll(t *testing.T, input string) ([]Token, error) {
	t.Helper()
	lexer := NewLexer([]byte(input))
	var tokens []Token
	for {
		tok, err := lexer.NextToken()
		if err != nil {
			return tokens, err
		}
		tokens = append(tokens, tok)
		if tok.Type == TokenEOF {
			return tokens, nil
		}
	}
}

func TestTokenTypeString(t *testing.T) {
	cases := []struct {
		typ  TokenType
		want string
	}{
		{TokenEOF, "end of input"},
		{TokenLeftBrace, "{"},
		{TokenRightBrace, "}"},
		{TokenLeftBracket, "["},
		{TokenRightBracket, "]"},
		{TokenColon, ":"},
		{TokenComma, ","},
		{TokenString, "string"},
		{TokenNumber, "number"},
		{TokenTrue, "true"},
		{TokenFalse, "false"},
		{TokenNull, "null"},
		{TokenIllegal, "illegal token"},
		{TokenType(200), "illegal token"},
	}
	for _, c := range cases {
		if got := c.typ.String(); got != c.want {
			t.Errorf("TokenType(%d).String() = %q, want %q", c.typ, got, c.want)
		}
	}
}

func TestNewLexer(t *testing.T) {
	lexer := NewLexer([]byte("{}"))
	if lexer == nil {
		t.Fatal("NewLexer returned nil")
	}
	pos := lexer.position()
	if pos.Line != 1 || pos.Column != 1 || pos.Offset != 0 {
		t.Errorf("initial position = %+v, want Line:1 Column:1 Offset:0", pos)
	}
}

func TestNextTokenPunctuation(t *testing.T) {
	tokens, err := lexAll(t, "{}[]:,")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []TokenType{
		TokenLeftBrace, TokenRightBrace,
		TokenLeftBracket, TokenRightBracket,
		TokenColon, TokenComma, TokenEOF,
	}
	if len(tokens) != len(want) {
		t.Fatalf("got %d tokens, want %d", len(tokens), len(want))
	}
	for i, wt := range want {
		if tokens[i].Type != wt {
			t.Errorf("token[%d] = %v, want %v", i, tokens[i].Type, wt)
		}
	}
}

func TestNextTokenKeywords(t *testing.T) {
	cases := []struct {
		input string
		typ   TokenType
	}{
		{"true", TokenTrue},
		{"false", TokenFalse},
		{"null", TokenNull},
	}
	for _, c := range cases {
		tokens, err := lexAll(t, c.input)
		if err != nil {
			t.Fatalf("%q: unexpected error: %v", c.input, err)
		}
		if tokens[0].Type != c.typ {
			t.Errorf("%q: type = %v, want %v", c.input, tokens[0].Type, c.typ)
		}
		if tokens[0].Value != c.input {
			t.Errorf("%q: value = %q, want %q", c.input, tokens[0].Value, c.input)
		}
	}
}

func TestNextTokenInvalidKeyword(t *testing.T) {
	if _, err := lexAll(t, "tru"); err == nil {
		t.Error("expected error for invalid keyword, got nil")
	}
	if _, err := lexAll(t, "nulk"); err == nil {
		t.Error("expected error for invalid keyword, got nil")
	}
}

func TestNextTokenStrings(t *testing.T) {
	cases := []struct {
		input string
		want  string
	}{
		{`"hello"`, "hello"},
		{`""`, ""},
		{`"with \"escaped\" quotes"`, `with "escaped" quotes`},
		{`"tab\tand\nnewline"`, "tab\tand\nnewline"},
		{`"unicode \u00e9"`, "unicode é"},
	}
	for _, c := range cases {
		tokens, err := lexAll(t, c.input)
		if err != nil {
			t.Fatalf("%q: unexpected error: %v", c.input, err)
		}
		if tokens[0].Type != TokenString {
			t.Errorf("%q: type = %v, want TokenString", c.input, tokens[0].Type)
		}
		if tokens[0].Value != c.want {
			t.Errorf("%q: value = %q, want %q", c.input, tokens[0].Value, c.want)
		}
	}
}

func TestNextTokenStringErrors(t *testing.T) {
	cases := []string{
		`"unterminated`,
		"\"line\nbreak\"",
		"\"ctrl\x01char\"",
		`"bad escape \`,
	}
	for _, c := range cases {
		if _, err := lexAll(t, c); err == nil {
			t.Errorf("%q: expected error, got nil", c)
		}
	}
}

func TestNextTokenNumbers(t *testing.T) {
	cases := []string{
		"0", "-0", "42", "-42", "3.14", "-3.14",
		"1e10", "1E10", "1e+10", "1e-10", "2.5e3", "0.5",
	}
	for _, c := range cases {
		tokens, err := lexAll(t, c)
		if err != nil {
			t.Fatalf("%q: unexpected error: %v", c, err)
		}
		if tokens[0].Type != TokenNumber {
			t.Errorf("%q: type = %v, want TokenNumber", c, tokens[0].Type)
		}
		if tokens[0].Value != c {
			t.Errorf("%q: value = %q, want %q", c, tokens[0].Value, c)
		}
	}
}

func TestNextTokenNumberErrors(t *testing.T) {
	cases := []string{
		"01",    // leading zero
		"-",     // lone minus
		"1.",    // no digit after decimal
		"1e",    // no digit in exponent
		"1e+",   // no digit after exponent sign
		"1.2.3", // trailing junk is not a number error but a separate token; checked below
	}
	// The first five are hard lexer errors.
	for _, c := range cases[:5] {
		if _, err := lexAll(t, c); err == nil {
			t.Errorf("%q: expected error, got nil", c)
		}
	}
}

func TestNextTokenWhitespaceAndPositions(t *testing.T) {
	tokens, err := lexAll(t, "  {\n  }")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tokens[0].Type != TokenLeftBrace {
		t.Fatalf("first token = %v, want TokenLeftBrace", tokens[0].Type)
	}
	if tokens[0].Position.Line != 1 || tokens[0].Position.Column != 3 {
		t.Errorf("'{' position = %+v, want Line:1 Column:3", tokens[0].Position)
	}
	if tokens[1].Position.Line != 2 {
		t.Errorf("'}' line = %d, want 2", tokens[1].Position.Line)
	}
}

func TestNextTokenUnexpectedCharacter(t *testing.T) {
	if _, err := lexAll(t, "@"); err == nil {
		t.Error("expected error for '@', got nil")
	}
}

func TestNextTokenNonASCII(t *testing.T) {
	if _, err := lexAll(t, "é"); err == nil {
		t.Error("expected error for non-ASCII start byte, got nil")
	}
}

func TestNextTokenEmptyInput(t *testing.T) {
	tokens, err := lexAll(t, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tokens) != 1 || tokens[0].Type != TokenEOF {
		t.Errorf("empty input tokens = %+v, want single EOF", tokens)
	}
}

func TestDigit(t *testing.T) {
	for b := byte('0'); b <= '9'; b++ {
		if !digit(b) {
			t.Errorf("digit(%q) = false, want true", b)
		}
	}
	for _, b := range []byte{'a', '/', ':', ' ', 0} {
		if digit(b) {
			t.Errorf("digit(%q) = true, want false", b)
		}
	}
}
