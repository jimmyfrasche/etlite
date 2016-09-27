//Package savepoint models the savepoint stack and transaction support.
package savepoint

import (
	"errors"
	"fmt"
)

//Stack models the savepoint stack and its interaction with transactions.
type Stack struct {
	trans bool
	s     []string
}

//New creates a new Stack.
func New() *Stack {
	return &Stack{}
}

//InTransaction if Stack has a transaction.
func (s *Stack) InTransaction() bool {
	return s.trans
}

//HasSavepoints if there are any savepoints on the Stack.
func (s *Stack) HasSavepoints() bool {
	return len(s.s) > 0
}

//Open reports whether InTransaction or HasSavepoints.
func (s *Stack) Open() bool {
	return s.InTransaction() || s.HasSavepoints()
}

//Top returns the outermost savepoint name, if HasSavepoints.
func (s *Stack) Top() string {
	return s.s[0]
}

//Begin a transaction.
func (s *Stack) Begin() error {
	if s.trans {
		return errors.New("cannot nest transactions")
	}
	if len(s.s) > 0 {
		return fmt.Errorf("cannot begin transaction with open savepoint stack %v", s.s)
	}
	s.trans = true
	return nil
}

//End a transaction.
func (s *Stack) End() error {
	if !s.trans {
		return errors.New("no open transaction to commit")
	}
	s.trans = false
	s.s = s.s[:0]
	return nil
}

//Savepoint adds a new savepoint to the Stack.
func (s *Stack) Savepoint(sp string) {
	s.s = append(s.s, sp)
}

//Release a savepoint from the Stack.
func (s *Stack) Release(sp string) error {
	p := s.last(sp)
	if p < 0 {
		return fmt.Errorf("attempting to release unknown savepoint %s", sp)
	}
	s.s = s.s[:len(s.s)-p]
	return nil
}

func (s *Stack) last(sp string) int {
	for i := len(s.s) - 1; i >= 0; i-- {
		if s.s[i] == sp {
			return i
		}
	}
	return -1
}
