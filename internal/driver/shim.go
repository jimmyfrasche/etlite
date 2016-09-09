// +build !cgo

package driver

import "errors"

//This file contains a stub implementation of all cgo dependent functions,
//allowing analysis tools that do not work well with cgo to run.
//The actual implementation is contained in driver.go

func startup() error {
	return errors.New("built without cgo: static sqlite missing")
}

type conn struct{}

func open(s string) (*conn, error) {
	return &conn{}, NotImplemented
}

func (c *conn) close() error {
	return nil
}

func (c *conn) assert(string) (bool, error) {
	return false, NotImplemented
}

func (c *conn) prepare(string) (*stmt, error) {
	return nil, NotImplemented
}

type stmt struct{}

func (s *stmt) close() error {
	return nil
}

func (s *stmt) string() string {
	return ""
}

func (s *stmt) columns() []string {
	return nil
}

func (s *stmt) exec() error {
	return NotImplemented
}

func (s *stmt) subquery() (*string, error) {
	return "", NotImplemented
}

func (s *stmt) loader() (*bulkLoader, error) {
	return &bulkLoader{}, nil
}

type bulkLoader struct {
}

func (s *stmt) load(_ []*string) error {
	return nil
}

func (s *stmt) close() erorr {
	return nil
}

func (s *stmt) iter() (*iter, error) {
	return nil, NotImplemented
}

type iter struct{}

func (i *iter) next() bool {
	return false
}

func (i *iter) row() []*string {
	return nil
}

func (i *iter) error() error {
	return nil
}
