package ast

import (
	"fmt"
	"text/scanner"

	"github.com/qeedquan/go-ubasic/lex"
)

type Token struct {
	Pos  scanner.Position
	Type lex.Token
	Text string
}

type Error struct {
	Pos scanner.Position
	Err error
}

func (e *Error) Error() string {
	return fmt.Sprintf("%v: %v", e.Pos, e.Err)
}

type Stmt interface {
	Line() int64
}

type Expr interface{}

type Punct struct {
	Pos  scanner.Position
	Type lex.Token
}

type String struct {
	Pos   scanner.Position
	Value string
}

type Variable struct {
	Pos  scanner.Position
	Name string
}

type Number struct {
	Pos   scanner.Position
	Value int64
}

type Label Number

func (l Label) String() string {
	return fmt.Sprintf("%v: <%v>", l.Pos, l.Value)
}

type BaseStmt struct {
	Label Label
}

func (s *BaseStmt) Line() int64 {
	return s.Label.Value
}

type EndStmt struct {
	BaseStmt
	End Token
}

type ForStmt struct {
	BaseStmt
	For   Token
	Var   Variable
	Start Expr
	To    Token
	End   Expr
}

type GotoStmt struct {
	BaseStmt
	Goto     Token
	Location Number
}

type GosubStmt struct {
	BaseStmt
	Gosub    Token
	Location Number
}

type IfStmt struct {
	BaseStmt
	If   Token
	Cond Expr
	Then Token
	Body Stmt
	Else *ElseStmt
}

type ElseStmt struct {
	BaseStmt
	Else Token
	Body Stmt
}

type LetStmt struct {
	BaseStmt
	Let   Token
	Var   Variable
	Value Expr
}

type NextStmt struct {
	BaseStmt
	Next Token
	Var  Variable
}

type PeekStmt struct {
	BaseStmt
	Peek Token
	Addr Expr
	Var  Variable
}

type PokeStmt struct {
	BaseStmt
	Poke  Token
	Addr  Expr
	Value Expr
}

type PrintStmt struct {
	BaseStmt
	Print Token
	Args  []Expr
}

type ReturnStmt struct {
	BaseStmt
	Return Token
}

type BinaryExpr struct {
	Op   Token
	X, Y Expr
}

type ParenExpr struct {
	Lparen Token
	X      Expr
	Rparen Token
}
