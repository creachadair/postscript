// Package code defines an interface for specifying PostScript program
// fragments.  A fragment defines a stack shape and has the ability to write
// itself in source format to an output stream.
package code

import (
	"bytes"
	"encoding/ascii85"
	"fmt"
	"io"
	"strconv"
	"strings"
)

// A Program represents one or more sequential instructions having a particular
// effect on the operand stack.
type Program interface {
	// Stack reports the stack signature of the program, consisting of the net
	// number of elements consumed from the stack (in) and the net number of
	// elements added to the stack (out).
	Stack() (in, out int)

	io.WriterTo
}

// A Seq is a sequential composition of programs.
type Seq []Program

func (s Seq) Stack() (in, out int)               { return seqStack(s) }
func (s Seq) WriteTo(w io.Writer) (int64, error) { return writeSeq(w, "", "", s...) }

// An Array constructs an array of the values of the given programs.
type Array []Program

func (a Array) Stack() (in, out int)               { return seqStack(a) }
func (a Array) WriteTo(w io.Writer) (int64, error) { return writeSeq(w, "[", "]", a...) }

// A Proc is a deferred-execution program.
type Proc []Program

func (p Proc) Stack() (in, out int)               { return 0, 1 }
func (p Proc) WriteTo(w io.Writer) (int64, error) { return writeSeq(w, "{", "}", p...) }

// An Int is an integer constant Program.
type Int int

func (z Int) Stack() (int, int)                  { return 0, 1 }
func (z Int) WriteTo(w io.Writer) (int64, error) { return writeString(w, strconv.Itoa(int(z))) }

// A Real is a floating-point constant Program.
type Real float64

func (r Real) Stack() (int, int) { return 0, 1 }

func (r Real) WriteTo(w io.Writer) (int64, error) {
	s := strconv.FormatFloat(float64(r), 'g', -1, 64)
	if !strings.Contains(s, ".") && !strings.Contains(s, "e") {
		s += "." // real literals must have a decimal or an exponent
	}
	return writeString(w, s)
}

// A String is a string constant Program.
type String string

func (s String) Stack() (int, int) { return 0, 1 }

var quoteMap = map[byte]byte{
	'\r': 'r', '\t': 't', '\b': 'b', '\f': 'f', '\\': '\\', '(': '(', ')': ')',
}

func (s String) WriteTo(w io.Writer) (int64, error) {
	var buf bytes.Buffer
	buf.WriteByte('(')
	for i := 0; i < len(s); i++ {
		b := s[i]
		if q, ok := quoteMap[b]; ok {
			buf.WriteByte('\\')
			buf.WriteByte(q)
		} else if b >= 32 && b < 128 || b == '\n' {
			buf.WriteByte(b)
		} else {
			fmt.Fprintf(&buf, "\\%03o", b)
		}
	}
	buf.WriteByte(')')
	return buf.WriteTo(w)
}

// A Name is a quoted name, whose execution is deferred.
type Name string

func (q Name) Stack() (int, int)                  { return 0, 1 }
func (q Name) WriteTo(w io.Writer) (int64, error) { return writeString(w, "/"+string(q)) }

// Bytes is a binary data constant Program.
type Bytes []byte

func (b Bytes) Stack() (int, int) { return 0, 1 }

func (b Bytes) WriteTo(w io.Writer) (int64, error) {
	enc := make([]byte, ascii85.MaxEncodedLen(len(b)))
	n := ascii85.Encode(enc, []byte(b))
	var buf bytes.Buffer
	buf.WriteString("<~")
	for i := 0; i < n; i += 72 {
		if i+80 < n {
			buf.WriteByte('\n')
			buf.Write(enc[i : i+72])
			continue
		}
		if i > 0 {
			buf.WriteByte('\n')
		}
		buf.Write(enc[i:n])
		if i > 0 {
			buf.WriteByte('\n')
		}
	}
	buf.WriteString("~>")
	return buf.WriteTo(w)
}

// Op constructs an operator with the specified name and stack signature.
func Op(name string, in, out int) Program { return op{name: name, in: in, out: out} }

type op struct {
	name    string
	in, out int
}

func (o op) Stack() (in, out int)               { return o.in, o.out }
func (o op) WriteTo(w io.Writer) (int64, error) { return writeString(w, o.name) }

// A UserOp represents a named, user-defined operator.
type UserOp struct {
	Op   Program // the operator to evaluate
	Defn Defn    // the definition of the operation
}

// Define constructs an operator definition from the specified program and a
// definition program to bind it.
func Define(name string, ps Program) UserOp {
	p := []Program{ps}
	if proc, ok := ps.(Proc); ok {
		p = []Program(proc)
	}
	in, out := seqStack(p)
	return UserOp{
		Op: Op(name, in, out),
		Defn: Defn{
			Name:  name,
			Value: ps,
		},
	}
}

// A Def is a program fragment that binds a name to a program.
type Defn struct {
	Name  string
	Value Program
}

func (d Defn) Stack() (in, out int)               { return 0, 0 }
func (d Defn) WriteTo(w io.Writer) (int64, error) { return writeSeq(w, "/"+d.Name+" ", " def", d.Value) }

// If is a conditional expression.
type If struct {
	Test Program
	Then Proc
	Else Proc
}

func (f If) Stack() (in, out int) {
	in, out = f.Test.Stack()
	out-- // the operator pops the test result

	tin, tout := seqStack(f.Then)
	fin, fout := seqStack(f.Else)

	// The input stack must be large enough to accommodate either arm of the
	// conditional. However, the output size may differ between the two. For a
	// conservative estimate, predict that the shorter branch will be taken.

	if f.Else != nil {
		if f.Then == nil || fin > tin {
			tin = fin // take the larger input requirement
		}
		if f.Then == nil || fout < tout {
			tout = fout // assert the smaller output size
		}
	}
	if tin > out {
		t := tin - out
		in += t
		out += t
	}
	return in, (out - tin) + tout
}

func (f If) WriteTo(w io.Writer) (int64, error) {
	seq := Seq{f.Test, f.Then}
	op := "if"
	if f.Then == nil {
		seq[1] = Proc{} // generate an empty proc
	}
	if f.Else != nil {
		seq = append(seq, f.Else)
		op = "ifelse"
	}
	seq = append(seq, Op(op, 0, 0))
	return seq.WriteTo(w)
}

// With evaluates a program in the scope of a user dictionary.
type With struct {
	Dict Program
	Body Program
}

func (b With) Stack() (in, out int) {
	in, out = b.Dict.Stack()
	out-- // begin consumes the dictionary
	bin, bout := b.Body.Stack()
	return merge(in, out, bin, bout)
}

func (b With) WriteTo(w io.Writer) (int64, error) {
	return writeSeq(w, "", "", b.Dict, Begin, b.Body, End)
}

func merge(ain, aout, bin, bout int) (in, out int) {
	if bin > aout {
		t := bin - aout
		ain += t
		aout += t
	}
	return ain, (aout - bin) + bout
}

// seqStack computes the stack signature of a sequential composition.
func seqStack(ps []Program) (in, out int) {
	if len(ps) == 0 {
		return
	}
	in, out = ps[0].Stack()
	pin, pout := seqStack(ps[1:])

	// If the tail of the sequence requires more stack than was left by the
	// first element, the overage must already exist prior to the composition.
	return merge(in, out, pin, pout)
}

// writeStringTo writes s to w, and adds the number of bytes written to the
// value of *total, returning any error from the write. If s == "", no write is
// invoked.
func writeStringTo(w io.Writer, total *int64, s string) error {
	if s == "" {
		return nil
	}
	nw, err := writeString(w, s)
	*total += nw
	return err
}

// writeString adapts io.WriteString to the signature of io.WriterTo.
func writeString(w io.Writer, s string) (int64, error) {
	nw, err := io.WriteString(w, s)
	return int64(nw), err
}

// writeSeq writes a sequential composition of programs, with an optional
// framing prefix and suffix.
func writeSeq(w io.Writer, pfx, sfx string, ps ...Program) (int64, error) {
	var total int64
	if err := writeStringTo(w, &total, pfx); err != nil {
		return total, err
	}
	for i, p := range ps {
		if i > 0 {
			if err := writeStringTo(w, &total, " "); err != nil {
				return total, err
			}
		}
		nw, err := p.WriteTo(w)
		total += nw
		if err != nil {
			return total, err
		}
	}
	err := writeStringTo(w, &total, sfx)
	return total, err
}
