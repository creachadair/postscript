package scanner

import (
	"io"
	"strings"
	"testing"
)

// TODO: Add tests for the Pos and End methods.

// scan runs a scanner on input and calls f for each successful s.Next.
func scan(t *testing.T, input string, f func(i int, s *Scanner)) {
	t.Helper()

	s := New(strings.NewReader(input))
	t.Logf("Scanning input %#q", input)
	for i := 0; s.Next() == nil; i++ {
		f(i, s)
	}
	if s.Err() != io.EOF {
		t.Errorf("After scanning: got %v, want EOF", s.Err())
	}
}

func TestRawTokens(t *testing.T) {
	tests := []struct {
		input string
		want  []string
	}{
		// Empty or all-whitespace inputs should produce no tokens.
		{"", nil},
		{"   ", nil},
		{"\t \t", nil},

		// Comments should include their terminator.
		{"% hello\n", []string{"% hello\n"}},

		// Somewhat unusually, comments can end with form-feed.
		{"% hello\f % world\n ", []string{"% hello\f", "% world\n"}},

		// Various name-shaped things.
		{"a /b //c $d ", []string{"a", "/b", "//c", "$d"}},
		{"-3\n2.5e9\n 2#1101", []string{"-3", "2.5e9", "2#1101"}},

		// Slashes should terminate name processing except at the start.
		{"eat/your//veggies", []string{"eat", "/your", "//veggies"}},

		// Self-delimiting names should delimit themselves.
		{"{a<<b>>c[d]}", []string{"{", "a", "<<", "b", ">>", "c", "[", "d", "]", "}"}},

		// String literals preserve whitespace inside them.
		{"(a\nb\nc d)", []string{"(a\nb\nc d)"}},

		// String literals respect balanced nested quotations, and unbalanced
		// nested quotations can be quoted. Note that at this point we have not
		// done any decoding so all the escapes are still there.
		{" (a (b c)\n d)\n", []string{"(a (b c)\n d)"}},
		{`(abc\(def)`, []string{`(abc\(def)`}},
		{`(\)\\\))`, []string{`(\)\\\))`}},

		// Hex and A85 literals.
		{"<66 6f 6f><~  AoDS  ~>", []string{"<66 6f 6f>", "<~  AoDS  ~>"}},
	}
	for _, test := range tests {
		scan(t, test.input, func(i int, s *Scanner) {
			got := s.Text()
			if i >= len(test.want) {
				t.Errorf("Extra token %d: %#q", i, got)
			} else if got != test.want[i] {
				t.Errorf("Token %d: got %#q, want %#q", i, got, test.want[i])
			}
		})
	}
}

func TestTokenTypes(t *testing.T) {
	tests := []struct {
		input string
		want  []Type
	}{
		{"", nil},
		{"/", []Type{QuotedName}},
		{"//", []Type{ImmediateName}},
		{"///", []Type{ImmediateName, QuotedName}},
		{" % ok\n all is /well", []Type{Comment, Name, Name, QuotedName}},
		{"//imm/o/lation", []Type{ImmediateName, QuotedName, QuotedName}},
		{"-.002 123 -98 16#FFFE", []Type{Real, Decimal, Decimal, Radix}},
		{"[alpha/bravo] % ok\n{charlie 1}", []Type{
			Name, Name, QuotedName, Name, Comment,
			Left, Name, Decimal, Right,
		}},
		{"(all\nyour\nbase)\n(are (belong)  to) us\n", []Type{
			LitString, LitString, Name,
		}},
		{"0.1e1 53 -19 5#110304 -9. 6.67E-19 5.e+3 0.0", []Type{
			Real, Decimal, Decimal, Radix, Real, Real, Real, Real,
		}},
	}
	for _, test := range tests {
		scan(t, test.input, func(i int, s *Scanner) {
			got := s.Type()
			if i >= len(test.want) {
				t.Errorf("Extra token %d: %v %#q", i, got, s.Text())
			} else if got != test.want[i] {
				t.Errorf("Token %d: got %v %#q, want %v", i, got, s.Text(), test.want[i])
			}
		})
	}
}

func TestString(t *testing.T) {
	tests := []struct {
		input string
		want  []string
	}{
		// String literals.
		{"(a (b) c)", []string{"a (b) c"}},
		{"(a \\(b c)", []string{"a (b c"}},
		{"  (a\n b c\\) def)", []string{"a\n b c) def"}},
		{"(ab\\\ncd\\\ref\\\r\ngh)", []string{"abcdefgh"}},

		// Hex literals.
		{"<> <  > <50> <5>", []string{"", "", "P", "P"}},
		{"<66 6f 6f>", []string{"foo"}},
		{"<32 31 3>", []string{"210"}},

		// A85 literals.
		{"<~~> <~  ~> <~ AoDS ~>", []string{"", "", "foo"}},

		// Names and punctuation.
		{"alpha/bravo charlie //xray", []string{"alpha", "bravo", "charlie", "xray"}},
		{"[full /plate (and) {packing}]<<steel>>", []string{
			"[", "full", "plate", "and", "{", "packing", "}", "]", "<<", "steel", ">>",
		}},

		// Comments.
		{"% foo\n% bar\f\n% baz\n ", []string{"foo", "bar", "baz"}},

		// Numbers.
		{"1.3 .0 2#1101 6.67e-11", []string{"1.3", ".0", "2#1101", "6.67e-11"}},
	}
	for _, test := range tests {
		scan(t, test.input, func(i int, s *Scanner) {
			got := s.String()
			if i >= len(test.want) {
				t.Errorf("Extra token %d: %v %#q", i, s.Type(), s.Text())
			} else if got != test.want[i] {
				t.Errorf("Token %d: got %v %#q, want %v", i, s.Type(), got, test.want[i])
			}
		})
	}
}

func TestTokenValues(t *testing.T) {
	scan(t, "-2 -1 0 1 2", func(i int, s *Scanner) {
		want := int64(i) - 2
		got, err := s.Int64()
		if err != nil {
			t.Errorf("Token %d [%#q]: Int64 failed: %v", i, s.Text(), err)
		} else if got != want {
			t.Errorf("Token %d: Int64: got %v, want %v", i, got, want)
		}
	})

	scan(t, "-2.0 -1 0.0 0.1e1 0.002e3", func(i int, s *Scanner) {
		want := float64(i) - 2
		got, err := s.Float64()
		if err != nil {
			t.Errorf("Token %d [%#q]: Float64 failed: %v", i, s.Text(), err)
		} else if got != want {
			t.Errorf("Token %d: Float64: got %v, want %v", i, got, want)
		}
	})
}

func TestScanErrors(t *testing.T) {
	tests := []string{
		// Unterminated string literals.
		`(unterminated string`,
		`(`,
		`<ac 9e 30`,
		`<`,
		`<~ apple pie `,
		`<~`,

		// Invalid contents.
		`< BOGUS HEX>`,
		`<~ xxx is not legal A85 xxx ~>`,
		`<~ all good so far oops ~ ~>`,
	}
	for _, test := range tests {
		s := New(strings.NewReader(test))
		for i := 0; s.Next() == nil; i++ {
			t.Logf("Token %d: %v %#q", i, s.Type(), s.Text())
		}
		if err := s.Err(); err == nil || err == io.EOF {
			t.Errorf("Scanning %#q: got %v, wanted failure", test, err)
		} else {
			t.Logf("Scanning %#q: got %v [OK]", test, err)
		}
	}
}
