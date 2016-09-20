package parse

import (
	"fmt"
	"io"
	"strconv"

	"github.com/qeedquan/go-ubasic/ast"
	"github.com/qeedquan/go-ubasic/lex"
)

type Parser struct {
	lex  *lex.Tokenizer
	look []ast.Token
	tok  ast.Token

	label ast.Label
	let   ast.Token
}

func NewParser(lex *lex.Tokenizer) *Parser {
	p := &Parser{
		lex: lex,
	}
	p.next()
	return p
}

func (p *Parser) Reset() {
	p.look = p.look[:0]
	p.label = ast.Label{}
	p.let = ast.Token{}
	p.next()
}

func (p *Parser) errf(format string, args ...interface{}) {
	err := &ast.Error{p.tok.Pos, fmt.Errorf(format, args...)}
	p.synch()
	panic(err)
}

func (p *Parser) synch() {
	for {
		p.next()
		if p.tok.Type == lex.CR || p.tok.Type == lex.EOF {
			p.next()
			return
		}
	}
}

func (p *Parser) next() {
	if len(p.look) > 0 {
		p.tok = p.look[0]
		p.look = p.look[1:]
		return
	}

	for {
		p.tok.Pos, p.tok.Type, p.tok.Text = p.lex.Next()
		if p.tok.Type != lex.REM {
			break
		}
	}
}

func (p *Parser) accept(typ lex.Token) ast.Token {
	if p.tok.Type != typ {
		p.errf("expected %q, but got %q", typ, p.tok.Type)
	}
	xtok := p.tok
	p.next()
	return xtok
}

func (p *Parser) acceptNumber() ast.Number {
	t := p.accept(lex.NUMBER)
	n, err := strconv.ParseInt(t.Text, 0, 64)
	if err != nil {
		p.errf("invalid number %q: %v", p.tok.Text, err)
	}

	return ast.Number{
		Pos:   t.Pos,
		Value: n,
	}
}

func (p *Parser) acceptVariable() ast.Variable {
	t := p.accept(lex.VARIABLE)
	return ast.Variable{
		Pos:  t.Pos,
		Name: t.Text,
	}
}

func (p *Parser) acceptCR() {
	if p.tok.Type == lex.CR {
		p.accept(lex.CR)
		return
	}
	if p.tok.Type != lex.EOF {
		p.errf("expected newline, got %q", p.tok.Type)
	}
}

func (p *Parser) Line() (stmt ast.Stmt, err error) {
	defer func() {
		if e := recover(); e != nil {
			err = e.(error)
		}
	}()

	switch p.tok.Type {
	case lex.EOF:
		return nil, io.EOF
	case lex.ERROR:
		p.errf(p.tok.Text)
		panic("unreachable")
	default:
		return p.stmt(), nil
	}
}

func (p *Parser) skipcr() {
	for p.tok.Type == lex.CR {
		p.next()
	}
}

func (p *Parser) stmt() ast.Stmt {
	p.skipcr()

	p.label = ast.Label(p.acceptNumber())
	p.let = ast.Token{}
	cr := true

	var s ast.Stmt
	switch p.tok.Type {
	case lex.PRINT:
		s = p.print()
	case lex.IF:
		s = p.if_()
		cr = false
	case lex.GOTO:
		s = p.goto_()
	case lex.GOSUB:
		s = p.gosub()
	case lex.RETURN:
		s = p.return_()
	case lex.FOR:
		s = p.for_()
	case lex.PEEK:
		s = p.peek()
	case lex.POKE:
		s = p.poke()
	case lex.NEXT:
		s = p.next_()
	case lex.END:
		s = p.end()
	case lex.LET:
		p.let = p.accept(lex.LET)
		fallthrough
	case lex.VARIABLE:
		s = p.let_()
	default:
		p.errf("unsupported statement %q", p.tok.Text)
	}
	if cr {
		p.acceptCR()
	}

	return s
}

func (p *Parser) print() *ast.PrintStmt {
	s := &ast.PrintStmt{}
	s.Label = p.label
	s.Print = p.accept(lex.PRINT)

loop:
	for {
		switch p.tok.Type {
		case lex.STRING:
			lit, err := strconv.Unquote(p.tok.Text)
			if err != nil {
				p.errf("invalid string %q: %v", p.tok.Text, err)
			}
			s.Args = append(s.Args, ast.String{p.tok.Pos, lit})
			p.next()
		case lex.COMMA, lex.SEMICOLON:
			s.Args = append(s.Args, ast.Punct{p.tok.Pos, p.tok.Type})
			p.next()
		case lex.VARIABLE:
			s.Args = append(s.Args, p.expr())
		case lex.NUMBER:
			s.Args = append(s.Args, p.expr())
		case lex.CR, lex.EOF:
			break loop
		default:
			p.errf("unknown print type %q", p.tok.Text)
		}
	}

	return s
}

func (p *Parser) if_() *ast.IfStmt {
	s := &ast.IfStmt{}
	s.Label = p.label
	s.If = p.accept(lex.IF)
	s.Cond = p.relation()
	s.Then = p.accept(lex.THEN)
	p.acceptCR()
	s.Body = p.stmt()

	tok := p.tok
	num := p.acceptNumber()
	if p.tok.Type == lex.ELSE {
		else_ := p.accept(lex.ELSE)
		p.acceptCR()
		body := p.stmt()

		s.Else = &ast.ElseStmt{
			BaseStmt: ast.BaseStmt{
				Label: ast.Label(num),
			},
			Else: else_,
			Body: body,
		}
	} else {
		p.look = []ast.Token{p.tok}
		p.tok = tok
	}

	return s
}

func (p *Parser) relation() ast.Expr {
	r1 := p.expr()
loop:
	for {
		switch op := p.tok; op.Type {
		case lex.LT, lex.GT, lex.LEQ, lex.GEQ, lex.NEQ, lex.EQ:
			p.next()
			r2 := p.expr()
			r1 = &ast.BinaryExpr{
				Op: op,
				X:  r1,
				Y:  r2,
			}
		default:
			break loop
		}
	}
	return r1
}

func (p *Parser) goto_() *ast.GotoStmt {
	s := &ast.GotoStmt{}
	s.Label = p.label
	s.Goto = p.accept(lex.GOTO)
	s.Location = p.acceptNumber()
	return s
}

func (p *Parser) gosub() *ast.GosubStmt {
	s := &ast.GosubStmt{}
	s.Label = p.label
	s.Gosub = p.accept(lex.GOSUB)
	s.Location = p.acceptNumber()
	return s
}

func (p *Parser) for_() *ast.ForStmt {
	s := &ast.ForStmt{}
	s.Label = p.label
	s.For = p.accept(lex.FOR)
	s.Var = p.acceptVariable()
	p.accept(lex.EQ)
	s.Start = p.expr()
	s.To = p.accept(lex.TO)
	s.End = p.expr()
	return s
}

func (p *Parser) peek() *ast.PeekStmt {
	s := &ast.PeekStmt{}
	s.Label = p.label
	s.Peek = p.accept(lex.PEEK)
	s.Addr = p.expr()
	p.accept(lex.COMMA)
	s.Var = p.acceptVariable()
	return s
}

func (p *Parser) poke() *ast.PokeStmt {
	s := &ast.PokeStmt{}
	s.Label = p.label
	s.Poke = p.accept(lex.POKE)
	s.Addr = p.expr()
	p.accept(lex.COMMA)
	s.Value = p.expr()
	return s
}

func (p *Parser) next_() *ast.NextStmt {
	s := &ast.NextStmt{}
	s.Label = p.label
	s.Next = p.accept(lex.NEXT)
	s.Var = p.acceptVariable()
	return s
}

func (p *Parser) end() *ast.EndStmt {
	s := &ast.EndStmt{}
	s.Label = p.label
	s.End = p.accept(lex.END)
	return s
}

func (p *Parser) let_() *ast.LetStmt {
	s := &ast.LetStmt{}
	s.Label = p.label
	s.Let = p.let
	s.Var = p.acceptVariable()
	p.accept(lex.EQ)
	s.Value = p.expr()
	return s
}

func (p *Parser) return_() *ast.ReturnStmt {
	s := &ast.ReturnStmt{}
	s.Label = p.label
	s.Return = p.accept(lex.RETURN)
	return s
}

func (p *Parser) expr() ast.Expr {
	t1 := p.term()
loop:
	for {
		switch op := p.tok; op.Type {
		case lex.PLUS, lex.MINUS, lex.AND, lex.OR:
			p.next()
			t2 := p.term()
			t1 = &ast.BinaryExpr{
				Op: op,
				X:  t1,
				Y:  t2,
			}
		default:
			break loop
		}
	}
	return t1
}

func (p *Parser) term() ast.Expr {
	f1 := p.factor()
loop:
	for {
		switch op := p.tok; op.Type {
		case lex.ASTR, lex.SLASH, lex.MOD:
			p.next()
			f2 := p.factor()
			f1 = &ast.BinaryExpr{
				Op: op,
				X:  f1,
				Y:  f2,
			}
		default:
			break loop
		}
	}
	return f1
}

func (p *Parser) factor() ast.Expr {
	var r ast.Expr
	switch p.tok.Type {
	case lex.NUMBER:
		r = p.acceptNumber()
	case lex.LPAREN:
		l := p.tok
		x := p.expr()
		r = &ast.ParenExpr{l, x, p.accept(lex.RPAREN)}
	default:
		r = p.acceptVariable()
	}
	return r
}
