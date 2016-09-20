package interp

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/qeedquan/go-ubasic/ast"
	"github.com/qeedquan/go-ubasic/lex"
	"github.com/qeedquan/go-ubasic/parse"
)

type Mach interface {
	io.Writer
	Peek(addr int64) int64
	Poke(addr, value int64)
}

type Stdio struct {
	Values map[int64]int64
}

func (Stdio) Write(b []byte) (int, error) { return os.Stdout.Write(b) }
func (s *Stdio) Peek(addr int64) int64    { return s.Values[addr] }
func (s *Stdio) Poke(addr, value int64)   { s.Values[addr] = value }

func NewStdio() *Stdio {
	return &Stdio{
		Values: make(map[int64]int64),
	}
}

type ForStack struct {
	Block int
	Var   string
	To    int64
}

type Interpreter struct {
	Mach Mach
	Halt bool
	PC   int

	Vars  map[string]int64
	Subs  []int
	Fors  []ForStack
	Locs  map[int64]int
	Lines []ast.Stmt
}

func NewInterpreter(mach Mach) *Interpreter {
	p := &Interpreter{
		Mach: mach,
		Locs: make(map[int64]int),
	}
	p.Reset()
	return p
}

func (p *Interpreter) Reset() {
	p.Halt = false
	p.PC = 0
	p.Vars = make(map[string]int64)
	p.Subs = p.Subs[:0]
	p.Fors = p.Fors[:0]
}

func (p *Interpreter) errf(format string, args ...interface{}) {
	panic(fmt.Errorf(format, args...))
}

func (p *Interpreter) Step() error {
	if p.PC >= len(p.Lines) {
		p.Halt = true
	}
	if p.Halt {
		return nil
	}

	s := p.Lines[p.PC]
	p.PC++
	return p.Eval(s)
}

func (p *Interpreter) Eval(s ast.Stmt) (err error) {
	defer func() {
		if e := recover(); e != nil {
			err = e.(error)
		}
	}()

	p.stmt(s)
	return
}

func (p *Interpreter) stmt(s ast.Stmt) {
	switch s := s.(type) {
	case *ast.ForStmt:
		p.for_(s)
	case *ast.NextStmt:
		p.next(s)
	case *ast.IfStmt:
		p.if_(s)
	case *ast.GotoStmt:
		p.goto_(s)
	case *ast.GosubStmt:
		p.gosub(s)
	case *ast.ReturnStmt:
		p.return_(s)
	case *ast.LetStmt:
		p.assign(s)
	case *ast.EndStmt:
		p.Halt = true
	case *ast.PeekStmt:
		p.Vars[s.Var.Name] = p.Mach.Peek(p.expr(s.Addr))
	case *ast.PokeStmt:
		p.Mach.Poke(p.expr(s.Addr), p.expr(s.Value))
	case *ast.PrintStmt:
		p.print(s)
	}

	return
}

func (p *Interpreter) for_(s *ast.ForStmt) {
	p.Vars[s.Var.Name] = p.expr(s.Start)
	p.Fors = append(p.Fors, ForStack{
		Block: p.PC,
		Var:   s.Var.Name,
		To:    p.expr(s.End),
	})
}

func (p *Interpreter) next(s *ast.NextStmt) {
	if n := len(p.Fors); n > 0 {
		f := &p.Fors[n-1]
		if f.Var == s.Var.Name {
			p.Vars[s.Var.Name]++
		}

		if p.Vars[s.Var.Name] <= f.To {
			p.PC = f.Block
		} else {
			p.Fors = p.Fors[:n-1]
		}
	} else {
		p.errf("%v: non-matching next", s.Label)
	}
}

func (p *Interpreter) if_(s *ast.IfStmt) {
	if p.expr(s.Cond) != 0 {
		p.stmt(s.Body)
	} else if s.Else != nil {
		p.stmt(s.Else.Body)
	}
}

func (p *Interpreter) goto_(s *ast.GotoStmt) {
	loc, found := p.Locs[s.Location.Value]
	if !found {
		p.errf("%v: goto: location %d does not exist", s.Label, s.Location.Value)
	}
	p.PC = loc
}

func (p *Interpreter) gosub(s *ast.GosubStmt) {
	p.Subs = append(p.Subs, p.PC)
	loc, found := p.Locs[s.Location.Value]
	if !found {
		p.errf("%v: gosub: location %d does not exist", s.Label, s.Location.Value)
	}
	p.PC = loc
}

func (p *Interpreter) return_(s *ast.ReturnStmt) {
	if len(p.Subs) == 0 {
		p.errf("%v: non-matching return", s.Label)
	}
	p.PC = p.Subs[len(p.Subs)-1]
	p.Subs = p.Subs[:len(p.Subs)-1]
}

func (p *Interpreter) assign(s *ast.LetStmt) {
	p.Vars[s.Var.Name] = p.expr(s.Value)
}

func (p *Interpreter) print(s *ast.PrintStmt) {
	w := p.Mach
	for _, arg := range s.Args {
		switch arg := arg.(type) {
		case *ast.BinaryExpr:
			fmt.Fprint(w, p.expr(arg))
		case *ast.ParenExpr:
			fmt.Fprint(w, p.expr(arg))
		case ast.String:
			fmt.Fprint(w, arg.Value)
		case ast.Variable:
			fmt.Fprint(w, p.expr(arg))
		case ast.Number:
			fmt.Fprint(w, p.expr(arg))
		case ast.Punct:
			switch arg.Type {
			case lex.COMMA:
				fmt.Fprint(w, " ")
			case lex.SEMICOLON:
			default:
				p.errf("%v: unknown print argument %T", s.Label, arg)
			}
		default:
			p.errf("%v: unknown print argument %T", s.Label, arg)
		}
	}
}

func truth(x bool) int64 {
	if x {
		return 1
	}
	return 0
}

func (p *Interpreter) expr(e ast.Expr) int64 {
	var n int64
	switch e := e.(type) {
	case *ast.BinaryExpr:
		l := p.expr(e.X)
		r := p.expr(e.Y)
		switch e.Op.Type {
		case lex.PLUS:
			n = l + r
		case lex.MINUS:
			n = l - r
		case lex.ASTR:
			n = l * r
		case lex.SLASH:
			n = l / r
		case lex.MOD:
			n = l % r
		case lex.AND:
			n = l & r
		case lex.OR:
			n = l | r
		case lex.XOR:
			n = l ^ r
		case lex.LT:
			n = truth(l < r)
		case lex.GT:
			n = truth(l > r)
		case lex.LEQ:
			n = truth(l <= r)
		case lex.GEQ:
			n = truth(l >= r)
		case lex.NEQ:
			n = truth(l != r)
		case lex.EQ:
			n = truth(l == r)
		default:
			p.errf("%v: unknown binary operator %q", e.Op.Pos, e.Op.Type)
		}
	case *ast.ParenExpr:
		n = p.expr(e.X)
	case ast.Variable:
		v, ok := p.Vars[e.Name]
		if !ok {
			p.errf("%v: unknown variable name %v", e.Pos, e.Name)
		}
		n = v
	case ast.Number:
		return e.Value
	}
	return n
}

func Run(mach Mach, name string, src []byte) error {
	var lexer lex.Tokenizer
	lexer.Init(lex.Config{}, name, src)
	parser := parse.NewParser(&lexer)
	interp := NewInterpreter(mach)

	for {
		line, err := parser.Line()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		interp.Lines = append(interp.Lines, line)
	}
	for i, s := range interp.Lines {
		interp.Locs[s.Line()] = i
	}

	interp.Reset()
	for !interp.Halt {
		err := interp.Step()
		if err != nil {
			return err
		}
	}

	return nil
}

func Repl(mach Mach, r io.Reader) error {
	var lexer lex.Tokenizer
	parser := parse.NewParser(&lexer)
	interp := NewInterpreter(mach)

	w := mach
	scan := bufio.NewScanner(r)

loop:
	for {
		fmt.Fprint(w, "> ")
		if !scan.Scan() {
			fmt.Fprintln(w)
			break
		}
		line := strings.TrimSpace(scan.Text())

		switch line {
		case "p":
			for _, s := range interp.Lines {
				fmt.Println(s)
			}
			continue loop

		case "q":
			break loop
		}

		lexer.Init(lex.Config{}, "", []byte(line))
		parser.Reset()
		stmt, err := parser.Line()
		if err == io.EOF || ek(err) {
			continue
		}

		addLine(interp, stmt)
		switch stmt.(type) {
		case *ast.GosubStmt:
			ek(replRun(interp))
		case *ast.GotoStmt:
			ek(replRun(interp))
		case *ast.NextStmt:
		case *ast.EndStmt:
		default:
			ek(interp.Eval(stmt))
		}
	}

	return nil
}

func replRun(p *Interpreter) error {
	p.PC = len(p.Lines) - 1
	for p.Halt = false; !p.Halt; {
		if err := p.Step(); err != nil {
			return err
		}
	}
	return nil
}

func addLine(p *Interpreter, s ast.Stmt) {
	if n, found := p.Locs[s.Line()]; found {
		p.Lines = append(p.Lines[:n], p.Lines[n+1:]...)
	}
	p.Lines = append(p.Lines, s)
	p.Locs = make(map[int64]int)
	for i, s := range p.Lines {
		p.Locs[s.Line()] = i
	}
	p.PC = len(p.Lines) - 1
}

func ek(err error) bool {
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return true
	}
	return false
}
