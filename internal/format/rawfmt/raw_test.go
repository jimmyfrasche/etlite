package rawfmt

import (
	"bufio"
	"io"
	"strings"
	"testing"
)

func slowjoin(tab, nl string, xss [][]string) string {
	var acc []string
	for _, xs := range xss {
		acc = append(acc, strings.Join(xs, tab))
	}
	return strings.Join(acc, nl)
}

var nls = []string{"\n", "\r\n"}

func nlType(nl string) string {
	if nl == "\r\n" {
		return "windows newline"
	}
	return "proper newline"
}

var matrix = [][]string{
	{"a", "b", "c"},
	{"1", "2", "3"},
	{"", "π", "long"},
}

func cmp(t *testing.T, ctx string, a, b [][]string) {
	if len(a) != len(b) {
		t.Fatalf("%s: Expected %d rows got %d", ctx, len(a), len(b))
	}
	for row := range a {
		if len(a[row]) != len(b[row]) {
			t.Fatalf("%s: Row %d expected %d cols got %d: %#v", ctx, row, len(a[row]), len(b[row]), b[row])
		}
		for col := range a[row] {
			if a[row][col] != b[row][col] {
				t.Fatalf("%s: Row %d col %d expected %q got %q", ctx, row, col, a[row][col], b[row][col])
			}
		}
	}
}

func newReader(s string) *bufio.Reader {
	return bufio.NewReader(strings.NewReader(s))
}

func fakeDecoder(nl, in string) *Decoder {
	return &Decoder{
		Tab:     '\t',
		UseCRLF: nl != "\n",
		r:       newReader(in),
	}
}

func dup(xs []string) []string {
	return append([]string(nil), xs...)
}

func TestDecoder(t *testing.T) {
	for _, nl := range nls {
		ctx := "decode " + nlType(nl)
		d := fakeDecoder(nl, slowjoin("\t", nl, matrix))
		//suffices to check read, everything else is slim wrappers around it
		var acc [][]string
		for rc := 0; ; rc++ {
			row, err := d.read()
			if err != nil && len(row) != 0 {
				t.Fatalf("%s: read %d got error %q and row %#v", ctx, rc, err, row)
			}
			if err == io.EOF {
				break
			}
			if err != nil {
				t.Fatalf("%s: read %d, unexpected error %q", ctx, rc, err)
			}
			acc = append(acc, dup(row))
		}
		cmp(t, ctx, matrix, acc)
	}
}
