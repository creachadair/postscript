package code

import (
	"bytes"
	"strings"
	"testing"
)

func TestConstants(t *testing.T) {
	tests := []struct {
		input Program
		want  string
	}{
		{Int(-5), "-5"},
		{Int(0), "0"},
		{Int(65534), "65534"},

		{Real(0), "0."},
		{Real(1), "1."},
		{Real(1e9), "1e+09"},
		{Real(6382.7e-23), "6.3827e-20"},

		{String(""), "()"},
		{String("cap the flag"), "(cap the flag)"},
		{String("hey (you)"), `(hey \(you\))`},
		{String("(a\nb\t(c)\n\nd)"), "(\\(a\nb\\t\\(c\\)\n\nd\\))"},
		{String("\x00\x01\x02"), `(\000\001\002)`},

		{Name("click"), "/click"},
		{Var("click"), "click"},

		{Bytes(nil), "<~~>"},
		{Bytes{}, "<~~>"},
		{Bytes("foo"), `<~AoDS~>`},

		// Long byte literals get chopped up into lines.
		{Bytes(strings.Repeat("APPLE", 20)), `<~
5u:BO76saH9LV6D:eX;D:f'hS5u:BO76saH9LV6D:eX;D:f'hS5u:BO76saH9LV6D:eX;D:f
'hS5u:BO76saH9LV6D:eX;D:f'hS5u:BO76saH9LV6D:eX;D:f'hS
~>`},
	}
	for _, test := range tests {
		runTest(t, test.input, 0, 1, test.want)
	}
}

func TestSequences(t *testing.T) {
	tests := []struct {
		input     Program
		win, wout int
		want      string
	}{
		{Op("foo", 1, 2), 1, 2, "foo"},
		{Op("foo", 0, 1), 0, 1, "foo"},
		{Seq{Int(1), Int(2), Add}, 0, 1, "1 2 add"},
		{Array{Int(1), Int(2), Add, Int(3)}, 0, 2, "[1 2 add 3]"},
		{Proc{Int(72), Mul}, 0, 1, "{72 mul}"},

		{If{
			Test: Seq{Dup, Int(25), Gt},
			Then: Proc{Int(25), Sub},
		}, 1, 1, "dup 25 gt {25 sub} if"},

		{If{
			Test: Seq{True},
			Then: Proc{Int(30), Cos, Mul},
			Else: Proc{Sin},
		}, 1, 1, "true {30 cos mul} {sin} ifelse"},

		{If{
			Test: Seq{False},
			Else: Proc{Mul},
		}, 2, 1, "false {} {mul} ifelse"},

		{Defn{"in", Proc{Int(72), Mul}}, 0, 0, "/in {72 mul} def"},
		{Defn{"hello", Proc{
			NewPath, Int(72), Int(144), MoveTo,
			Name("Helvetica"), FindFont, Int(12), ScaleFont, SetFont,
			String("Hello, World!\n"), Show,
		}}, 0, 0, "/hello {newpath 72 144 moveto /Helvetica findfont 12 " +
			"scalefont setfont (Hello, World!\n) show} def"},

		{With{
			Dict: Seq{Int(2), NDict},
			Body: Seq{True, Pop},
		}, 0, 0, "2 dict begin true pop end"},
	}
	for _, test := range tests {
		runTest(t, test.input, test.win, test.wout, test.want)
	}
}

func runTest(t *testing.T, p Program, win, wout int, want string) {
	t.Helper()

	gin, gout := p.Stack()
	if gin != win || gout != wout {
		t.Errorf("Stack [%+v]: got (%d, %d), want (%d, %d)", p, gin, gout, win, wout)
	}
	var buf bytes.Buffer
	if _, err := p.WriteTo(&buf); err != nil {
		t.Errorf("Write [%+v]: unexpected error: %v", p, err)
	} else if got := buf.String(); got != want {
		t.Errorf("Write [%+v]: got %#q, want %#q", p, got, want)
	}
}
