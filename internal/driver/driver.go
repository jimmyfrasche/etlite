// +build cgo

package driver

/*
#cgo CFLAGS: -std=gnu99

#cgo LDFLAGS: -lpthread -lpcre

#cgo linux   LDFLAGS: -ldl
#cgo windows LDFLAGS: -lmingwex -lmingw32

#cgo CFLAGS: -DSQLITE_THREADSAFE

#cgo CFLAGS: -DSQLITE_ENABLE_DBSTAT_VTAB
#cgo CFLAGS: -DSQLITE_ENABLE_SOUNDEX
#cgo CFLAGS: -DSQLITE_ENABLE_RTREE
#cgo CFLAGS: -DSQLITE_ENABLE_JSON1

#cgo LDFLAGS: -lm
#cgo CFLAGS: -DSQLITE_ENABLE_FTS5

#cgo LDFLAGS: -licuuc -licui18n
#cgo CFLAGS: -DSQLITE_ENABLE_ICU

//this lets the extensions defined in this directory be included statically
#cgo CFLAGS: -DSQLITE_CORE

#include <stdlib.h>

#include "sqlite3.h"

#include "ext.h"
#include "bind.h"
*/
import "C"

import (
	"errors"
	"unsafe"

	"github.com/jimmyfrasche/etlite/internal/internal/errint"
)

//bulkRowsAtOnce is how many rows we strive to handle at a time
//when doing bulk read/writes to sqlite.
var bulkRowsAtOnce = int(C.bulkRowsAtOnce)

func ok(rc C.int) bool {
	switch rc {
	case C.SQLITE_OK, C.SQLITE_ROW, C.SQLITE_DONE:
		return true
	}
	return false
}

func errstr(rc C.int) error {
	if ok(rc) {
		return nil
	}
	return errors.New(C.GoString(C.sqlite3_errstr(rc)))
}

func startup() error {
	return errstr(C.startup())
}

type conn struct {
	db *C.sqlite3
}

func open(name string) (*conn, error) {
	c := &conn{}

	nm := C.CString(name)
	defer C.free(unsafe.Pointer(nm))

	flags := C.int(C.SQLITE_OPEN_FULLMUTEX | C.SQLITE_OPEN_READWRITE | C.SQLITE_OPEN_CREATE)
	var db *C.sqlite3
	r := C.sqlite3_open_v2(nm, &db, flags, nil)
	if !ok(r) {
		return nil, errmsg(c.db)
	}
	c.db = db
	return c, nil
}

func errmsg(db *C.sqlite3) error {
	//TODO make this a method on conn
	return errors.New(C.GoString(C.sqlite3_errmsg(db)))
}

func (c *conn) close() error {
	if c == nil || c.db == nil {
		return nil
	}

	var err error
	r := C.sqlite3_close_v2(c.db)
	if !ok(r) {
		err = errmsg(c.db)
	}

	c.db = nil
	return err
}

func (c *conn) assert(query string) (bool, error) {
	if c == nil || c.db == nil {
		return false, errint.New("no database connection when asserting")
	}
	if len(query) == 0 {
		return false, errint.New("no query to assert on")
	}

	qln := C.int(len(query))
	q := C.CString(query) //TODO replace with string cache
	defer C.free(unsafe.Pointer(q))

	var out C.int

	ret := C.sqlbind_assert_query(c.db, q, qln, &out)
	if ret != C.SQLITE_OK || out < 0 {
		switch out {
		default:
			return false, errint.Newf("unexpected error code in assert: %d", out)
		case C.sqlbind_err_sqlite:
			return false, errmsg(c.db)
		case C.sqlbind_err_num_cols:
			return false, misuse("assert query must have exactly one column")
		case C.sqlbind_err_no_result:
			return false, misuse("assert query must have exactly one result, none returned")
		case C.sqlbind_err_type:
			return false, misuse("assert query must return a boolean")
		case C.sqlbind_err_range:
			return false, misuse("assert query must return a boolean, got arbitrary integer")
		case C.sqlbind_err_too_many_results:
			return false, misuse("assert query must have exactly one result, multiple returned")
		}
	}

	return out != 1, nil
}

func (c *conn) prepare(query string) (*stmt, error) {
	if c == nil || c.db == nil {
		return nil, errint.New("no database connection when preparing statement")
	}
	s := &stmt{c: c}

	q := C.CString(query) //TODO replace with string cache
	defer C.free(unsafe.Pointer(q))

	// was using instead of -1: C.int(len(query)) + 1
	var p *C.sqlite3_stmt
	r := C.sqlite3_prepare_v2(c.db, q, -1, &p, nil)
	if !ok(r) {
		return nil, errmsg(c.db)
	}
	if p == nil {
		return nil, errors.New("no query specified")
	}
	s.p = p

	n := int(C.sqlite3_column_count(s.p))
	if n != 0 {
		s.cols = make([]string, n)
		for i := range s.cols {
			s.cols[i] = C.GoString(C.sqlite3_column_name(s.p, C.int(i)))
		}
	}

	s.binds = C.sqlite3_bind_parameter_count(s.p)

	return s, nil
}

type stmt struct {
	c     *conn
	p     *C.sqlite3_stmt
	binds C.int
	cols  []string
}

func (s *stmt) close() error {
	if s == nil || s.p == nil || s.c == nil {
		return nil
	}

	var err error
	r := C.sqlite3_finalize(s.p)
	if !ok(r) {
		err = errmsg(s.c.db)
	}

	s.c, s.p, s.cols = nil, nil, nil
	return err
}

func (s *stmt) string() string {
	//this is a pointer in s.p, do not free
	cs := C.sqlite3_sql(s.p)
	return C.GoString(cs)
}

func (s *stmt) columns() []string {
	if s == nil || s.p == nil || s.c == nil {
		return nil
	}

	return s.cols
}

func (s *stmt) exec() error {
	if s == nil || s.p == nil || s.c == nil || len(s.cols) != 0 {
		return errors.New("cannot exec malformed statement")
	}
	rv := C.sqlite3_step(s.p)
	if !ok(rv) {
		return errmsg(s.c.db)
	}
	if rv != C.SQLITE_DONE {
		return errors.New("exec called but statement not done")
	}
	return nil
}

func (s *stmt) subquery() (*string, error) {
	switch len(s.cols) {
	case 0:
		return nil, errint.New("attempted to use statement with no columns as subquery")
	case 1:
		//good, we can proceed
	default:
		return nil, errors.New("a subquery can only return a single result")
	}

	var cstr *C.char
	var length C.int
	rv := C.sqlbind_subquery(s.p, &cstr, &length)
	if !ok(rv) {
		return nil, errmsg(s.c.db)
	}

	if cstr == nil {
		return nil, nil
	}

	result := C.GoStringN(cstr, length)
	C.free(unsafe.Pointer(cstr))
	return &result, nil
}

//TODO this is from an earlier iteration of the design of this API.
//The new API is clumsily and hastily implemented in terms of it.
//It needs to be rewritten into the new API at some point.

func (s *stmt) bulkLoad(xs []*string) error {
	if s.binds == 0 {
		return errors.New("attempting to bulk load on statement without bound variables")
	}
	if len(xs) == 0 {
		return nil
	}

	//convert xs into char**
	L := C.int(len(xs))
	ptrsz := C.size_t(unsafe.Sizeof((*C.char)(nil)))
	arr := (**C.char)(C.malloc(ptrsz * C.size_t(L)))
	view := (*[1 << 30]*C.char)(unsafe.Pointer(arr))[:len(xs):len(xs)]
	for i, x := range xs {
		if x == nil {
			view[i] = nil
		} else {
			view[i] = C.CString(*x)
		}
	}

	rv := C.sqlbind_bulk_insert(s.p, s.binds, arr, L)
	if !ok(rv) {
		return errmsg(s.c.db)
	}

	return nil
}

func (s *stmt) loader() (*bulkLoader, error) {
	binds := int(s.binds)
	if binds == 0 {
		return nil, errors.New("cannot create loader on statement without ? binds")
	}
	return &bulkLoader{
		s:   s,
		acc: make([]*string, 0, bulkRowsAtOnce*binds),
	}, nil
}

type bulkLoader struct {
	s *stmt
	n int //how many rows we've seen this pass

	//TODO refactor sqlbind_bulk_insert and replace this with
	//an allocated once C vector
	acc []*string
}

func (b *bulkLoader) flush() error {
	if b.n == 0 {
		return nil
	}
	//TODO replace with basically what's in b.s.bulkLoad but reusing the C vector
	err := b.s.bulkLoad(b.acc)
	b.acc = b.acc[:0]
	return err
}

func (b *bulkLoader) load(vs []*string) error {
	//TODO replace with "append" to C vec
	b.acc = append(b.acc, vs...)

	b.n++
	if b.n < bulkRowsAtOnce {
		return nil
	}

	return b.flush()
}

func (b *bulkLoader) close() error {
	if b == nil || b.s == nil {
		return nil
	}

	err := b.flush()
	b.s = nil
	b.acc = nil //TODO replace by freeing C vector allocated in loader
	return err
}

func (s *stmt) iter() (*iter, error) {
	if s == nil || s.p == nil {
		return nil, errors.New("cannot create iterator for malformed statement")
	}

	if s.binds != 0 {
		return nil, errors.New("cannot iterate over stored procedure with binds")
	}

	return &iter{c: s.c, s: s}, nil
}

type iter struct {
	c    *conn
	s    *stmt
	err  error
	done bool
	dat  []*string
}

func (i *iter) next() bool {
	if i == nil || i.err != nil || i.c == nil || i.s == nil || i.done {
		return false
	}
	var n C.int
	var ret **C.char
	rv := C.sqlbind_bulk_read(i.s.p, &ret, &n)
	if !ok(rv) {
		i.done = true
		i.err = errmsg(i.c.db)
		return false
	}
	if rv == C.SQLITE_DONE {
		i.done = true
		return false
	}
	defer C.free(unsafe.Pointer(ret))
	N := int(n)
	view := (*[1 << 30]*C.char)(unsafe.Pointer(ret))[:N:N]
	i.dat = make([]*string, N)
	for n, cs := range view {
		if cs != nil {
			s := C.GoString(cs)
			i.dat[n] = &s
		}
	}
	return true
}

func (i *iter) row() []*string {
	if i == nil || i.err != nil || i.c == nil || i.s == nil || i.done {
		return nil
	}
	return i.dat
}

func (i *iter) error() error {
	if i == nil || i.c == nil || i.s == nil {
		return nil
	}
	return i.err
}
