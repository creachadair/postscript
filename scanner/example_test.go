package scanner_test

import (
	"fmt"
	"strings"

	"bitbucket.org/creachadair/postscript/scanner"
)

func needsSpace(t scanner.Type) bool {
	switch t {
	case scanner.Decimal, scanner.Radix, scanner.Real, scanner.Name:
		return true
	}
	return false
}

func ExampleScanner() {
	s := scanner.New(strings.NewReader(`% Draw a line
/in { 72 mul } def
/mt { moveto } def
newpath
  .5 in .5 in mt   % use half inch margins
  8 in 10.5 in mt
  10 setlinewidth
  0 setgray        % black
stroke % and that's all!`))

	// Use the scanner to strip out comments and unnecessary whitespace, and
	// wrap the source to 80 columns.
	col := 0
	var prev scanner.Type
	for s.Next() == nil {
		t := s.Type()
		if t == scanner.Comment {
			continue
		}
		if col+len(s.Text()) > 80 {
			fmt.Print("\n")
			col = 0
		} else if needsSpace(prev) && needsSpace(t) {
			fmt.Print(" ")
			col++
		}

		prev = t
		n, _ := fmt.Print(s.Text())
		col += n
	}
	// Output:
	// /in{72 mul}def/mt{moveto}def newpath .5 in .5 in mt 8 in 10.5 in mt 10
	// setlinewidth 0 setgray stroke
}
