// +build cgo

package driver

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"
	"testing"
)

func TestMain(m *testing.M) {
	flag.Parse()

	//Need to initialize before any test.
	if err := Init(); err != nil {
		log.Print("could not init binding")
		log.Fatal(err)
	}

	//If this doesn't work, nothing will so check as long as we're here.
	c, err := Open(":memory:")
	if err != nil {
		log.Print("could not open :memory:")
		log.Fatal(err)
	}
	if err = c.Close(); err != nil {
		log.Print("could not close :memory:")
		log.Fatal(err)
	}

	os.Exit(m.Run())
}

func with(t *testing.T, f func(*Conn)) {
	c, err := Open(":memory:")
	if err != nil {
		t.Fatal("could not create db")
	}

	f(c)

	if err := c.Close(); err != nil {
		t.Fatal("could not destroy db")
	}
}

//TestPrepare tests preparing a statement and reading its results.
func TestPrepare(t *testing.T) {
	const q = "SELECT 1 AS one, 2 AS two, 3 AS three"
	with(t, func(c *Conn) {
		s, err := c.Prepare(q)
		if err != nil {
			t.Fatal("could not prepare statement:", err)
		}

		if str := s.String(); str != q {
			t.Log("could not reflect query string")
			t.Log("expected:", q)
			t.Fatal("got:", str)
		}

		if s.binds != 0 {
			t.Fatal("should be no bind parameters, got:", s.binds)
		}

		cols := s.Columns()
		if len(cols) != 3 {
			t.Fatal("expected three columns, got:", len(cols))
		}

		for i, c := range []string{"one", "two", "three"} {
			if cols[i] != c {
				t.Fatal("column", i, "expected:", c, "got:", cols[i])
			}
		}

		if err := s.Close(); err != nil {
			t.Fatal("could not close statement, got:", err)
		}
	})
}

func s(r string) *string {
	return &r
}

var table = [][]*string{
	{s("1"), nil, s("squirrel")},
	{s("2"), s("two"), nil},
	{s("3"), nil, nil},
	{s("4"), s("avocado"), s("σ∈⁵ℝ«⌋")},
}

func fmts(s *string) string {
	if s == nil {
		return "<nil>"
	}
	return strconv.Quote(*s)
}

func fmtvec(xs []*string) string {
	s := "["
	for i, x := range xs {
		s += fmts(x)
		if i != len(xs)-1 {
			s += ", "
		}
	}
	s += "]"
	return s
}

func cmp(xs, ys []*string) error {
	if len(xs) != len(ys) {
		return fmt.Errorf("|xs|=%d but |ys|=%d", len(xs), len(ys))
	}

	for i, x := range xs {
		y := ys[i]
		if x == nil && y == nil {
			continue
		}
		if x == nil && y != nil || y == nil && x != nil || *x != *y {
			return fmt.Errorf("expected: %s but got: %s", fmts(x), fmts(y))
		}
	}

	return nil
}

//TestBind tests
//	- exec (creating a table)
//	- load (populating a table)
//	- iter (reading results back out)
//which covers all of the major points.
func TestBind(t *testing.T) {
	with(t, func(c *Conn) {
		open := func(q string) *Stmt {
			s, err := c.Prepare(q)
			if err != nil {
				t.Fatalf("could not prepare %q, got: %s\n", q, err)
			}
			return s
		}
		cleanup := func(which string, s *Stmt) {
			if err := s.Close(); err != nil {
				t.Fatal("could not close", which, "statement, got:", err)
			}
		}

		create := open("CREATE TABLE t (a INT, b TEXT, c TEXT)") //TODO test with c BLOB
		defer cleanup("create", create)
		if err := create.Exec(); err != nil {
			t.Fatal("could not exec create table, got:", err)
		}

		load := open("INSERT INTO t VALUES (?, ?, ?)")
		defer cleanup("load", load)

		loader, err := load.Loader()
		if err != nil {
			t.Fatal("could not create loader, got:", err)
		}
		for i, row := range table {
			if err := loader.Load(row); err != nil {
				t.Fatalf("could not load row %d, got: %s", i, err)
			}
		}
		if err := loader.Close(); err != nil {
			t.Fatal("could not close loader, got:", err)
		}

		read := open("SELECT * FROM t")
		defer cleanup("read", read)

		iter, err := read.Iter()
		if err != nil {
			t.Fatal("could not create iterator, got:", err)
		}
		i := 0
		for iter.Next() {
			if err := cmp(table[i], iter.Row()); err != nil {
				t.Log("Reading row", i)
				t.Log("Expected:", fmtvec(table[i]))
				t.Log("Got:", fmtvec(iter.Row()))
				t.Fatal(err)
			}
			i++
		}
		if err := iter.Err(); err != nil {
			t.Fatal("iterator reported:", err)
		}
		if i != len(table) {
			t.Fatal("expected", len(table), "rows, got:", i)
		}
	})
}

func TestAssert(t *testing.T) {
	t.Error("TODO")
}

func TestSubquery(t *testing.T) {
	t.Error("TODO")
}
