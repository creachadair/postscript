package code_test

import (
	"fmt"
	"os"

	"bitbucket.org/creachadair/postscript/code"
)

func ExampleProgram() {
	inch := code.Define("inch", code.Seq{
		code.Int(72), code.Mul,
	})
	s := code.Seq{
		inch.Def,
		code.NewPath,
		code.Int(1), inch.Op, code.Int(1), inch.Op, code.MoveTo,
		code.String("Hello, world!"),
		code.Show,
	}
	s.WriteTo(os.Stdout)
	in, out := s.Stack()
	fmt.Println("\nin:", in, "out:", out)
	// Output:
	// /inch {72 mul} def newpath 1 inch 1 inch moveto (Hello, world!) show
	// in: 0 out: 0
}
