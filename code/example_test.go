package code_test

import (
	"fmt"
	"os"

	"bitbucket.org/creachadair/postscript/code"
)

func ExampleProgram() {
	inch := code.Define("inch", code.Proc{
		code.Int(72), code.Mul,
	})
	s := code.Seq{
		inch.Defn,
		code.NewPath,
		code.Int(1), inch.Op, code.Int(1), inch.Op, code.MoveTo,
		code.Name("Helvetica"), code.FindFont,
		code.Int(16), code.ScaleFont, code.SetFont,
		code.String("Hello, world!"),
		code.Show,
		code.ShowPage,
	}
	s.WriteTo(os.Stdout)
	in, out := s.Stack()
	fmt.Println("\nin:", in, "out:", out)
	// Output:
	// /inch {72 mul} def newpath 1 inch 1 inch moveto /Helvetica findfont 16 scalefont setfont (Hello, world!) show showpage
	// in: 0 out: 0
}
