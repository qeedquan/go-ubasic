package lex

import (
	"fmt"
	"strings"
	"text/scanner"
	"unicode"
	"unicode/utf8"
)

type Config struct {
	ScanComments bool
}

type Tokenizer struct {
	conf Config
	name string
	src  []byte

	ch                 rune
	offset             int
	rdOffset           int
	line, lastLine     int
	column, lastColumn int
}

func (t *Tokenizer) Init(conf Config, name string, src []byte) {
	*t = Tokenizer{
		conf:   conf,
		name:   name,
		src:    src,
		line:   1,
		column: 1,
	}
	t.next()
}

const (
	eof = -1
)

func (t *Tokenizer) next() {
	t.lastLine = t.line
	t.lastColumn = t.column
	if t.rdOffset < len(t.src) {
		t.offset = t.rdOffset
		if t.ch == '\n' {
			t.line++
			t.column = 1
		}
		r, w := utf8.DecodeRune(t.src[t.rdOffset:])
		t.rdOffset += w
		t.ch = r
	} else {
		t.offset = len(t.src)
		if t.ch == '\n' {
			t.line++
			t.column = 1
		}
		t.ch = eof
	}
}

func (t *Tokenizer) Next() (pos scanner.Position, tok Token, lit string) {
scan:
	t.skipws()

	pos = scanner.Position{
		Filename: t.name,
		Offset:   t.offset,
		Line:     t.lastLine,
		Column:   t.lastColumn,
	}
	switch ch := t.ch; {
	case isLetter(ch):
		lit = t.ident()
		tok = lookupIdent(lit)
		if tok == REM {
			lit += t.comment()
			if !t.conf.ScanComments {
				goto scan
			}
		}
	case unicode.IsDigit(ch):
		tok, lit = t.number()
	case ch == '"':
		tok, lit = t.string()
	case ch == eof:
		tok = EOF
	default:
		t.next()

		lit = string(ch)
		switch ch {
		case '\n', '\r':
			tok = CR
		case ',':
			tok = COMMA
		case ';':
			tok = SEMICOLON
		case '<':
			tok = LT
			if t.ch == '=' {
				tok = LEQ
				lit = "<="
				t.next()
			}
		case '>':
			tok = GT
			if t.ch == '=' {
				tok = GEQ
				lit = ">="
				t.next()
			}
		case '!':
			if t.ch == '=' {
				tok = NEQ
				lit = "!="
				t.next()
			}
		case '=':
			tok = EQ
		case '(':
			tok = LPAREN
		case ')':
			tok = RPAREN
		case '^':
			tok = XOR
		case '&':
			tok = AND
		case '|':
			tok = OR
		case '+':
			tok = PLUS
		case '-':
			tok = MINUS
		case '*':
			tok = ASTR
		case '/':
			tok = SLASH
		case '%':
			tok = MOD
		case '#':
			tok = HASH
		default:
			tok = ERROR
			lit = fmt.Sprintf("unknown character %q", ch)
		}
	}
	return
}

func (t *Tokenizer) skipws() {
	for t.ch == '\t' || t.ch == ' ' {
		t.next()
	}
}

func (t *Tokenizer) comment() string {
	offs := t.offset
	for t.ch != '\n' {
		t.next()
	}
	t.next()
	return string(t.src[offs:t.offset])
}

func isLetter(ch rune) bool {
	return ch == '_' || unicode.IsLetter(ch)
}

func isDigit(ch rune) bool {
	return '0' <= ch && ch <= '9'
}

func (t *Tokenizer) ident() string {
	offs := t.offset
	for isLetter(t.ch) || isDigit(t.ch) {
		t.next()
	}
	return string(t.src[offs:t.offset])
}

func lookupIdent(ident string) Token {
	switch strings.ToLower(ident) {
	case "let":
		return LET
	case "print":
		return PRINT
	case "if":
		return IF
	case "then":
		return THEN
	case "else":
		return ELSE
	case "for":
		return FOR
	case "to":
		return TO
	case "next":
		return NEXT
	case "goto":
		return GOTO
	case "gosub":
		return GOSUB
	case "return":
		return RETURN
	case "call":
		return CALL
	case "rem":
		return REM
	case "peek":
		return PEEK
	case "poke":
		return POKE
	case "end":
		return END
	default:
		return VARIABLE
	}
}

func (t *Tokenizer) number() (Token, string) {
	offs := t.offset
	for isDigit(t.ch) {
		t.next()
	}
	return NUMBER, string(t.src[offs:t.offset])
}

func (t *Tokenizer) string() (Token, string) {
	offs := t.offset
	for {
		t.next()
		if t.ch == eof || t.ch == '\r' || t.ch == '\n' {
			return ERROR, "unterminated string"
		}
		if t.ch == '"' {
			break
		}
	}
	t.next()
	return STRING, string(t.src[offs:t.offset])
}
