// Program psmin removes comments and unnecessary whitespace from
// PostScript source text.
package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"

	"github.com/creachadair/postscript/scanner"
)

func init() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, `
Usage: %[1]s               # read from stdin
       %[1]s filename...   # read from files

Read PostScript source text from the specified files, and write
equivalent code to stdout without comments or unnecessary spaces.

With no arguments, read from stdin.
Use "-" as a filename to read from stdin among other files.
If multiple files are named, their contents are concatenated.

`, filepath.Base(os.Args[0]))
		flag.PrintDefaults()
	}
}

func main() {
	flag.Parse()

	args := flag.Args()
	if len(args) == 0 {
		args = append(args, "-")
	}
	for _, name := range args {
		var err error
		if name == "-" {
			err = scan(os.Stdout, os.Stdin)
		} else if f, oerr := os.Open(name); err != nil {
			log.Fatalf("Input: %v", oerr)
		} else {
			err = scan(os.Stdout, f)
		}
		if err != nil {
			log.Fatalf("Processing %q: %v", name, err)
		}
	}
}

func scan(w io.Writer, r io.ReadCloser) error {
	defer r.Close()
	s := scanner.New(r)

	var last scanner.Type
	for s.Next() == nil {
		cur := s.Type()
		if cur == scanner.Comment {
			continue
		}
		if scanner.NeedSpaceBetween(last, cur) {
			io.WriteString(w, " ")
		}
		io.WriteString(w, s.Text())
		last = cur
	}
	io.WriteString(w, "\n")
	if s.Err() == io.EOF {
		return nil
	}
	return s.Err()
}
