package translate

import (
	"bytes"
	"fmt"
	"io"
	"strings"

	"mvdan.cc/sh/syntax"
)

// Translator
//
// The translation functions internally panic, which gets caught by File
type Translator struct {
	buf bytes.Buffer
}

func NewTranslator() *Translator {
	return &Translator{}
}

func (t *Translator) WriteTo(w io.Writer) (int64, error) {
	return t.buf.WriteTo(w)
}

func (t *Translator) File(f *syntax.File) (err error) {
	// So I don't have to write if err all the time
	defer func() {
		if v := recover(); v != nil {
			if perr, ok := v.(error); ok {
				err = perr
				return
			}
			panic(v)
		}
	}()

	for _, stmt := range f.Stmts {
		t.stmt(stmt)
		t.str("\n")
	}

	for _, comment := range f.Last {
		t.comment(&comment)
	}

	return nil
}

func (t *Translator) stmt(s *syntax.Stmt) {
	for _, comment := range s.Comments {
		t.comment(&comment)
	}

	t.command(s.Cmd)
}

func (t *Translator) command(c syntax.Command) {
	switch c := c.(type) {
	case *syntax.ArithmCmd:
		unsupported(c)
	case *syntax.BinaryCmd:
		t.binaryCmd(c)
	case *syntax.Block:
		unsupported(c)
	case *syntax.CallExpr:
		t.callExpr(c)
	case *syntax.CaseClause:
		unsupported(c)
	case *syntax.CoprocClause:
		unsupported(c)
	case *syntax.DeclClause:
		t.declClause(c)
	case *syntax.ForClause:
		unsupported(c)
	case *syntax.FuncDecl:
		unsupported(c)
	case *syntax.IfClause:
		unsupported(c)
	case *syntax.LetClause:
		unsupported(c)
	case *syntax.Subshell:
		unsupported(c)
	case *syntax.TestClause:
		unsupported(c)
	case *syntax.TimeClause:
		unsupported(c)
	case *syntax.WhileClause:
		unsupported(c)
	default:
		unsupported(c)
	}
}

func (t *Translator) binaryCmd(c *syntax.BinaryCmd) {
	switch c.Op {
	case syntax.AndStmt:
		unsupported(c)
	case syntax.OrStmt:
		unsupported(c)
	case syntax.Pipe:
	case syntax.PipeAll:
		unsupported(c)
	}
	unsupported(c)
}

func (t *Translator) callExpr(c *syntax.CallExpr) {
	if len(c.Args) == 0 {
		// assignment
		for n, a := range c.Assigns {
			if n > 0 {
				t.str("; ")
			}

			t.printf("set %s ", a.Name.Value)
			t.word(a.Value)
		}
	} else {
		// call
		if len(c.Assigns) > 0 {
			t.str("env ")
			for _, a := range c.Assigns {
				if a.Value == nil {
					t.printf("-u %s ", a.Name.Value)
				} else {
					t.printf("%s=", a.Name.Value)
					t.word(a.Value)
					t.str(" ")
				}
			}
		}

		for i, a := range c.Args {
			if i > 0 {
				t.str(" ")
			}
			t.word(a)
		}
	}
}

func (t *Translator) declClause(c *syntax.DeclClause) {
	var prefix string
	if c.Variant != nil {
		switch c.Variant.Value {
		case "export":
			prefix = " -gx"
		case "local":
			prefix = " -l"
		default:
			unsupported(c)
		}
	}

	for _, a := range c.Assigns {
		t.printf("set%s %s ", prefix, a.Name.Value)
		if a.Value == nil {
			t.printf("$%s", a.Name.Value)
		} else {
			t.word(a.Value)
		}
	}
}

func (t *Translator) word(w *syntax.Word) {
	for _, part := range w.Parts {
		t.wordPart(part)
	}
}

var specialVariables = map[string]string{
	"?": "$status",
}

func (t *Translator) wordPart(wp syntax.WordPart) {
	switch wp := wp.(type) {
	case *syntax.Lit:
		t.str(wp.Value)
	case *syntax.SglQuoted:
		t.escapedString(wp.Value)
	case *syntax.DblQuoted:
		for _, part := range wp.Parts {
			switch part := part.(type) {
			case *syntax.Lit:
				t.escapedString(part.Value)
			default:
				t.wordPart(part)
			}
		}
	case *syntax.ParamExp:
		t.paramExp(wp)
	case *syntax.CmdSubst:
		t.str("(")
		for i, s := range wp.Stmts {
			if i > 0 {
				t.str("; ")
			}
			t.stmt(s)
		}
		t.str(")")
	case *syntax.ArithmExp:
		unsupported(wp)
	case *syntax.ProcSubst:
		unsupported(wp)
	case *syntax.ExtGlob:
		unsupported(wp)
	default:
		unsupported(wp)
	}
}

func (t *Translator) paramExp(p *syntax.ParamExp) {
	if p.Short {
		t.printf("$%s", p.Param.Value)
		return
	}
	if !p.Excl && !p.Length && !p.Width {
		t.printf("{$%s}", p.Param.Value)
		return
	}
	unsupported(p)
}

var stringReplacer = strings.NewReplacer("\\", "\\\\", "'", "\\'")

func (t *Translator) escapedString(literal string) {
	t.str("'")
	stringReplacer.WriteString(&t.buf, literal)
	t.str("'")
}

func (t *Translator) comment(c *syntax.Comment) {
	t.printf("#%s\n", c.Text)
}

func (t *Translator) str(s string) {
	t.buf.WriteString(s)
}

func (t *Translator) printf(format string, arg ...interface{}) {
	fmt.Fprintf(&t.buf, format, arg...)
}