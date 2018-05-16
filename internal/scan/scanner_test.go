package scan

import (
	"strings"
	"testing"
)

func TestScannerNumber(t *testing.T) {
	data := []struct {
		Value string
		Want  rune
	}{
		{Value: "+99", Want: Uint},
		{Value: "42", Want: Int},
		{Value: "0", Want: Int},
		{Value: "-17", Want: Int},
		{Value: "1_000", Want: Int},
		{Value: "5_349_221", Want: Int},
		{Value: "0xDEADBEEF", Want: Int},
		{Value: "0xdeadbeef", Want: Int},
		{Value: "0xdead_beef", Want: Int},
		{Value: "0o01234567", Want: Int},
		{Value: "0o755", Want: Int},
		{Value: "0b11010110", Want: Int},
		{Value: "+1.0", Want: Float},
		{Value: "3.1415", Want: Float},
		{Value: "-0.01", Want: Float},
		{Value: "5e+22", Want: Float},
		{Value: "1e6", Want: Float},
		{Value: "-2E-2", Want: Float},
		{Value: "6.626e-34", Want: Float},
		{Value: "9_224_617.445_991_228_313", Want: Float},
	}
	for i, d := range data {
		s := NewScanner(strings.NewReader(d.Value))
		if r := s.Scan(); r != d.Want {
			t.Errorf("%d) parsing %q failed! want %q got %q", i+1, d.Value, TokenString(d.Want), TokenString(r))
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
	for i, d := range data {
		s := NewScanner(strings.NewReader(d.Value))
		if r := s.Scan(); r != d.Want {
			t.Errorf("%d) parsing %q failed! want %q got %q", i+1, d.Value, TokenString(d.Want), TokenString(r))
		}
		if s.Text() != d.Value {
			t.Errorf("%d) scanning failed! want %s, got %s", i+1, d.Value, s.Text())
		}
	}
}
