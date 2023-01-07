// Package scanner implements a lexical scanner for PostScript.
package scanner

import (
	"bufio"
	"bytes"
	"encoding/ascii85"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"
)

// A Scanner consumes PostScript tokens from an input stream.  Use the Next
// method to parse tokens sequentially from the input.
type Scanner struct {
	input    *bufio.Reader // the unconsumed input
	text     *bytes.Buffer // the text of the current token
	err      error         // the last non-nil error reported
	token    Type          // the type of the current token
	pos, end int
}

// Type denotes the lexical type of a token.
type Type int

// The legal values for a token type.
const (
	Invalid       Type = iota // an invalid token
	Comment                   // a comment: % foo
	LitString                 // a string literal: (foo)
	HexString                 // a hex literal: <666f6f>
	A85String                 // an ascii85 literal: <~AoDS~>
	Decimal                   // a decimal integer: 25
	Radix                     // a radix integer: 2#1101
	Real                      // a floating-point value: -6.3e2
	Name                      // a name: foo
	QuotedName                // a quoted name: /foo
	ImmediateName             // an immediate name: //foo
	Left                      // a left bracket: {
	Right                     // a right bracket: }

	numTypes
)

// New constructs a *Scanner that reads from r.
func New(r io.Reader) *Scanner {
	return &Scanner{
		input: bufio.NewReader(r),   // unconsumed input
		text:  bytes.NewBuffer(nil), // the current token's text
	}
}

var (
	// Floating-point notation: -.002 34.5 -3.62 123.6e10 1.0E-5 1E6 -1. 0.0
	numReal = regexp.MustCompile(`^-?(\d+([eE][-+]?\d+)|(\d*\.\d+|\d+\.)([eE][-+]?\d+)?)$`)

	// Signed decimal integer notation: 123 -98 43445 0 +17
	numInteger = regexp.MustCompile(`^[-+]?\d+$`)

	// Radix notation: 8#1777 16#FFFE 2#1000
	// This expression allows invalid radix/digit combinations.
	numRadix = regexp.MustCompile(`^\d+#[0-9A-Za-z]+$`)

	// TODO: It's not clear how, or whether, sign is supposed to be accepted on
	// radix numbers.  I followed ghostscript here in omitting it, but the
	// reference manual is not explicit on the topic.
)

// Next advances s to the next token in the stream and returns nil if a valid
// token is available. If no further tokens are available, it returns io.EOF;
// otherwise it reports what went wrong.
func (s *Scanner) Next() error {
	// Reset state
	s.text.Reset()
	s.pos = s.end
	s.token = Invalid
	s.err = nil

	for {
		b, err := s.byte()
		if err != nil {
			return s.seterr(err)
		} else if isSpace(b) {
			s.pos = s.end
			continue // skip whitespace
		}

		s.text.WriteByte(b)
		switch b {
		case '%':
			return s.scanComment()

		case '(':
			return s.scanString()

		case '[', ']':
			s.token = Name // self-delimiting names
			return nil

		case '{':
			s.token = Left
			return nil
		case '}':
			s.token = Right
			return nil

		case '<':
			// This might be different things, depending on what follows.
			c, err := s.byte()
			if err == io.EOF {
				return s.seterr(errors.New("unterminated hex string"))
			} else if err != nil {
				return s.seterr(err)
			} else if c == '~' { // ascii85 literal
				s.text.WriteByte(c)
				return s.scanA85()
			}
			s.unget()
			if c != '<' { // hex literal
				return s.scanHex()
			}
			return s.scanNamelike(b)

		default:
			return s.scanNamelike(b)
		}
	}
}

// Err returns the last error reported by Next.
func (s *Scanner) Err() error { return s.err }

// Type reports the lexical type of the current token.
func (s *Scanner) Type() Type { return s.token }

// Text returns the literal text of the current token, or "".
func (s *Scanner) Text() string { return s.text.String() }

// Pos returns the starting byte offset of the current token in the input.
func (s *Scanner) Pos() int { return s.pos }

// End returns the ending byte offset of the current token in the input.
func (s *Scanner) End() int { return s.end }

// ErrInvalidFormat is reported when decoding a token value that does not match
// the specified result format.
var ErrInvalidFormat = errors.New("invalid format")

// Int64 returns the value of the current token as an int64. If the token
// cannot be converted to an integer value it returns 0, ErrInvalidFormat.
// Real tokens are truncated to an integer without error.
func (s *Scanner) Int64() (int64, error) {
	switch s.token {
	case Decimal:
		return strconv.ParseInt(s.Text(), 10, 64)
	case Real:
		f, err := strconv.ParseFloat(s.Text(), 64)
		if err != nil {
			return 0, err
		}
		return int64(f), nil
	case Radix:
		parts := strings.SplitN(s.Text(), "#", 2)
		r, err := strconv.Atoi(parts[0])
		if err != nil {
			return 0, err
		}
		v, err := strconv.ParseInt(parts[1], r, 64)
		if err != nil {
			return 0, err
		}
		return v, nil
	default:
		return 0, ErrInvalidFormat
	}
}

// Float64 returns the value of the current token as a float64.  If the token
// cannot be converted to a floating-point value it returns 0, ErrInvalidFormat.
// Integer tokens value converted to float64 without error.
func (s *Scanner) Float64() (float64, error) {
	switch s.token {
	case Real:
		return strconv.ParseFloat(s.Text(), 64)
	case Decimal, Radix:
		z, err := s.Int64()
		if err != nil {
			return 0, err
		}
		return float64(z), nil
	default:
		return 0, ErrInvalidFormat
	}
}

// String returns the decoded value of the current token as a string. This has
// different effects depending on the type:
//
// Numeric tokens and punctuation are returned as-written.
//
// Name tokens are returned with quotes removed (that is, /name becomes name).
//
// String literals are stripped of their quotes and decoded.
//
// Comment tokens are stripped of all leading "%" as well as any leading and
// trailing whitespace that remains after doing so.
func (s *Scanner) String() string {
	switch s.token {
	case Decimal, Left, Name, Radix, Real, Right:
		return s.Text()
	case QuotedName:
		return strings.TrimPrefix(s.Text(), "/")
	case ImmediateName:
		return strings.TrimPrefix(s.Text(), "//")
	case Comment:
		return strings.TrimSpace(strings.TrimLeft(s.Text(), "%"))
	case LitString:
		text := s.Text()
		unquoted := text[1 : len(text)-1] // remove outer "(" and ")"
		return decodeLiteral(unquoted)
	case HexString:
		text := s.Text()
		unquoted := text[1 : len(text)-1] // remove outer "<" and ">"
		return decodeHex(unquoted)
	case A85String:
		text := s.Text()
		unquoted := text[2 : len(text)-2] // remove outer "<~" and "~>"
		return decodeA85(unquoted)
	default:
		return ""
	}
}

func (s *Scanner) seterr(err error) error {
	s.err = err
	return err
}

func (s *Scanner) byte() (byte, error) {
	b, err := s.input.ReadByte()
	if err == nil {
		s.end++
	}
	return b, err
}

func (s *Scanner) unget() {
	s.input.UnreadByte()
	s.end--
}

func (s *Scanner) scanComment() error {
	for {
		b, err := s.byte()
		if err == io.EOF {
			s.end++
		} else if err != nil {
			return s.seterr(err)
		} else {
			s.text.WriteByte(b)
		}
		if err == io.EOF || b == '\n' || b == '\f' {
			s.token = Comment
			return nil
		}
	}
}

func (s *Scanner) scanString() error {
	depth := 1   // the opening quote is already buffered
	esc := false // true when we saw a backslash
	for {
		b, err := s.byte()
		if err == io.EOF {
			return s.seterr(errors.New("unterminated string"))
		} else if err != nil {
			return s.seterr(err)
		}
		if b == '\\' {
			esc = !esc
		} else if esc {
			esc = false
		} else if b == '(' {
			depth++
		} else if b == ')' {
			depth--
		}
		s.text.WriteByte(b)
		if b == ')' && depth == 0 {
			s.token = LitString
			return nil
		}
	}
}

// scanHex reads a hex encoded string literal, assuming the leading quote has
// already been buffered.
func (s *Scanner) scanHex() error {
	for {
		b, err := s.byte()
		if err == io.EOF {
			return s.seterr(errors.New("unterminated hex string"))
		} else if err != nil {
			return s.seterr(err)
		}
		s.text.WriteByte(b)
		if b == '>' {
			s.token = HexString
			return nil
		} else if !isHex(b) && !isSpace(b) {
			return s.seterr(fmt.Errorf("invalid hex %c", b))
		}
	}
}

// scanA85 reads an ascii85 encoded string literal, assuming the leading quote
// has already been buffered.
func (s *Scanner) scanA85() error {
	for {
		b, err := s.byte()
		if err == io.EOF {
			return s.seterr(errors.New("unterminated ascii85 string"))
		} else if err != nil {
			return s.seterr(err)
		}
		s.text.WriteByte(b)
		if b == '~' {
			c, err := s.byte()
			if err != nil || c != '>' {
				return s.seterr(errors.New("invalid closing ascii85 quote"))
			}
			s.text.WriteByte('>')
			s.token = A85String
			return nil
		} else if !isA85(b) && !isSpace(b) {
			return s.seterr(fmt.Errorf("invalid ascii85 %c", b))
		}
	}
}

// scanNamelike reads and classifies a name or number token, whose first byte
// is already buffered.
func (s *Scanner) scanNamelike(first byte) error {
	// Ref: "Any token that consists entirely of regular characters and cannot
	// be interpreted as a number is treated as a name object. All characters
	// except delimiters and whitespace characters can appear in names,
	// including characters ordinarily considered to be punctuation."

	for {
		b, err := s.byte()
		if err == io.EOF {
			s.end++
			break
		} else if err != nil {
			return s.seterr(err)
		}

		// A name may begin with "/" or "//", but otherwise "/" ends a name.
		// The names "<<" and ">>" are special-cased to be self-delimiting in
		// language level 2 and higher.
		if first != 0 {
			ok := b == first
			first = 0
			if ok {
				s.text.WriteByte(b)
				if b != '/' {
					break // i.e., << or >>
				}
				continue
			}
		}

		// Any self-delimiting marker terminates the name.
		if isSpace(b) || isSpecial(b) {
			s.unget()
			break
		}
		s.text.WriteByte(b)
	}

	// Upon reaching this point we have a name or a number in the buffer, but we
	// aren't sure which kind. The remainder of the work is to classify it.
	switch text := s.text.String(); {
	case strings.HasPrefix(text, "//"):
		s.token = ImmediateName
	case strings.HasPrefix(text, "/"):
		s.token = QuotedName
	case numReal.MatchString(text):
		s.token = Real
	case numInteger.MatchString(text):
		s.token = Decimal
	case numRadix.MatchString(text):
		s.token = Radix
	default:
		s.token = Name
	}
	return nil
}

var quoteMap = map[byte]byte{
	'n': '\n', 'r': '\r', 't': '\t', 'b': '\b', 'f': '\f', '\\': '\\', '(': '(', ')': ')',
}

func decodeLiteral(s string) string {
	buf := bytes.NewBuffer(make([]byte, 0, len(s)))
	esc := false
	for i := 0; i < len(s); i++ {
		ch := s[i]
		if esc {
			esc = false
			r, ok := quoteMap[ch]
			if ok {
				// standard escapes (see quoteMap)
				ch = r
			} else if i+2 < len(s) && isOctal(s[i]) && isOctal(s[i+1]) && isOctal(s[i+2]) {
				// octal byte \ooo
				ch = 64*(s[i]-'0') + 8*(s[i+1]-'0') + 1*(s[i+2]-'0')
			} else if ch == '\r' {
				// CR or CRLF pair, to be folded out
				if i+1 < len(s) && s[i+1] == '\n' {
					i++ // skip the LF
				}
				continue
			} else if ch == '\n' {
				continue // LF to be folded out
			}
			// the escape is discharged
		} else if ch == '\\' {
			esc = true
			continue
		}
		buf.WriteByte(ch)
	}
	return buf.String()
}

func decodeHex(s string) string {
	var buf []byte

	var cur byte
	var odd bool
	for i := 0; i < len(s); i++ {
		if isHex(s[i]) {
			cur = 16*cur + hexVal(s[i])
			odd = !odd
			if !odd {
				buf = append(buf, cur)
				cur = 0
			}
		}
	}
	if odd {
		buf = append(buf, 16*cur) // x becomes x0
	}
	return string(buf)
}

func decodeA85(s string) string {
	buf := make([]byte, len(s))
	nw, _, _ := ascii85.Decode(buf, []byte(s), true) // flush
	return string(buf[:nw])
}

func isSpace(b byte) bool {
	// Table 3.1, White-space characters
	switch b {
	case '\x00', '\t', '\n', '\f', '\r', ' ':
		return true
	}
	return false
}

func isHex(b byte) bool {
	return b >= '0' && b <= '9' || b >= 'a' && b <= 'f' || b >= 'A' && b <= 'F'
}

func hexVal(b byte) byte {
	if b >= 'a' {
		return b - 'a' + 10
	} else if b >= 'A' {
		return b - 'A' + 10
	}
	return b - '0'
}

func isOctal(b byte) bool { return b >= '0' && b <= '7' }

func isA85(b byte) bool { return b >= '!' && b <= 'u' }

func isSpecial(b byte) bool {
	switch b {
	case '(', ')', '<', '>', '[', ']', '{', '}', '/', '%':
		return true
	}
	return false
}

// A mapping of pairs of token types that need whitespace to separate them.
// Given types x and y, spaces[x][y] == true if x followed by y requires space.
var spaces = [numTypes][numTypes]bool{
	Decimal:       {Decimal: true, Radix: true, Real: true, Name: true},
	Radix:         {Decimal: true, Radix: true, Real: true, Name: true},
	Real:          {Decimal: true, Radix: true, Real: true, Name: true},
	Name:          {Decimal: true, Radix: true, Real: true, Name: true},
	QuotedName:    {Decimal: true, Radix: true, Real: true, Name: true},
	ImmediateName: {Decimal: true, Radix: true, Real: true, Name: true},
}

// NeedSpaceBetween reports whether spaces are required between a token of type
// prev and a token of type next to preserve lexical structure.
func NeedSpaceBetween(prev, next Type) bool { return spaces[prev][next] }
