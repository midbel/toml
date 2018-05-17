package scan

import (
	"strings"
	"testing"
)

func TestScannerSeries(t *testing.T) {
	data := []struct {
		Value string
		Want  []rune
	}{
		{
			Value: "option",
			Want:  []rune{Ident},
		},
		{
			Value: "option = 3.14",
			Want:  []rune{Ident, Equal, Float},
		},
		{
			Value: "option = {x = 3.14, y = 1.3}",
			Want: []rune{
				Ident,
				Equal,
				LeftCurlyBracket,
				Ident,
				Equal,
				Float,
				Comma,
				Ident,
				Equal,
				Float,
				RightCurlyBracket,
			},
		},
		{
			Value: "option = {x = 3, y = 1}",
			Want: []rune{
				Ident,
				Equal,
				LeftCurlyBracket,
				Ident,
				Equal,
				Int,
				Comma,
				Ident,
				Equal,
				Int,
				RightCurlyBracket,
			},
		},
		{
			Value: "option = {name = \"midbel\", email=\"midbel@foobar.org\"}",
			Want: []rune{
				Ident,
				Equal,
				LeftCurlyBracket,
				Ident,
				Equal,
				String,
				Comma,
				Ident,
				Equal,
				String,
				RightCurlyBracket,
			},
		},
	}
	const err = "%d) fail to scan %q at position %d: want %s, got %q (prev: %s, %s)"
	var s Scanner
	for i, d := range data {
		s.Reset(strings.NewReader(d.Value))
		// s := NewScanner(strings.NewReader(d.Value))
		var (
			p   rune
			str string
		)
		for j, k := 0, s.Scan(); k != EOF; j, k = j+1, s.Scan() {
			if k != d.Want[j] {
				w, g, v := TokenString(d.Want[j]), TokenString(k), TokenString(p)
				t.Errorf(err, i+1, d.Value, j, w, g, v, str)
				break
			}
			p, str = d.Want[i], s.Text()
		}
	}
}

func TestScannerNumbers(t *testing.T) {
	data := []struct {
		Value string
		Want  rune
		Match bool
	}{
		{Value: "+99", Want: Uint},
		{Value: "42", Want: Int, Match: true},
		{Value: "0", Want: Int, Match: true},
		{Value: "-17", Want: Int, Match: true},
		{Value: "1_000", Want: Int, Match: true},
		{Value: "5_349_221", Want: Int},
		{Value: "0xDEADBEEF", Want: Int, Match: true},
		{Value: "0xdeadbeef", Want: Int, Match: true},
		{Value: "0xdead_beef", Want: Int, Match: true},
		{Value: "0o01234567", Want: Int, Match: true},
		{Value: "0o755", Want: Int, Match: true},
		{Value: "0b11010110", Want: Int, Match: true},
		{Value: "+1.0", Want: Float},
		{Value: "3.1415", Want: Float, Match: true},
		{Value: "-0.01", Want: Float, Match: true},
		{Value: "5e+22", Want: Float},
		{Value: "1e6", Want: Float, Match: true},
		{Value: "-2E-2", Want: Float, Match: true},
		{Value: "6.626e-34", Want: Float, Match: true},
		{Value: "9_224_617.445_991_228_313", Want: Float},
	}
	var s Scanner
	for i, d := range data {
		s.Reset(strings.NewReader(d.Value))
		if r := s.Scan(); r != d.Want {
			t.Errorf("%d) parsing %q failed! want %q got %q", i+1, d.Value, TokenString(d.Want), TokenString(r))
		}
		if d.Match && d.Value != s.Text() {
			t.Errorf("%d) scanning %q failed! got %q", i+1, d.Value, s.Text())
		}
	}
}

func TestScannerDateTime(t *testing.T) {
	data := []struct {
		Value string
		Want  rune
	}{
		{Value: "1979-05-27T07:32:00Z", Want: DateTime},
		{Value: "1979-05-27T00:32:00-07:00", Want: DateTime},
		{Value: "1979-05-27T00:32:00.999999-07:00", Want: DateTime},
		{Value: "1979-05-27 07:32:00Z", Want: DateTime},
		{Value: "1979-05-27T07:32:00", Want: DateTime},
		{Value: "1979-05-27T00:32:00.999999", Want: DateTime},
		{Value: "1979-05-27", Want: Date},
		{Value: "07:32:00", Want: Time},
		{Value: "00:32:00.999999", Want: Time},
	}
	var s Scanner
	for i, d := range data {
		s.Reset(strings.NewReader(d.Value))
		// s := NewScanner(strings.NewReader(d.Value))
		if r := s.Scan(); r != d.Want {
			t.Errorf("%d) parsing %q failed! want %q got %q", i+1, d.Value, TokenString(d.Want), TokenString(r))
		}
		if s.Text() != d.Value {
			t.Errorf("%d) scanning failed! want %s, got %s", i+1, d.Value, s.Text())
		}
	}
}
