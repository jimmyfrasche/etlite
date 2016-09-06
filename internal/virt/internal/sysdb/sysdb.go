//Package sysdb encapsulates the sys database ETLite creates before running.
package sysdb

import (
	"strings"

	"github.com/jimmyfrasche/etlite/internal/driver"
	"github.com/jimmyfrasche/etlite/internal/internal/errint"
)

const (
	attach    = `ATTACH ':memory:' AS sys`
	createEnv = `CREATE TABLE sys.env (
	name TEXT PRIMARY KEY ON CONFLICT REPLACE,
	value TEXT NOT NULL
) WITHOUT ROWID`
	createArg  = `CREATE TABLE sys.arg (value TEXT NOT NULL)`
	insEnv     = `INSERT INTO sys.env VALUES (?, ?)`
	insArg     = `INSERT INTO sys.arg VALUES (?)`
	readAllEnv = `SELECT name || '=' || value FROM sys.env`
)

//Sysdb represents the sys db in etlite.
type Sysdb struct {
	readAllEnv *driver.Stmt
}

//New create a sys db populated with args and env.
func New(conn *driver.Conn, args, env []string) (sys *Sysdb, err error) {
	for _, s := range [...]string{attach, createEnv, createArg} {
		if err = exec(conn, s); err != nil {
			return nil, err
		}
	}

	err = load(conn, insArg, args, func(arg string) ([]*string, error) {
		return []*string{&arg}, nil
	})
	if err != nil {
		return nil, err
	}

	err = load(conn, insEnv, env, func(e string) ([]*string, error) {
		key, value, err := splitEnv(e)
		if err != nil {
			return nil, err
		}
		return []*string{&key, &value}, nil
	})
	if err != nil {
		return nil, err
	}

	sys = &Sysdb{}
	sys.readAllEnv, err = conn.Prepare(readAllEnv)
	if err != nil {
		return nil, errint.Wrap(err)
	}

	return sys, nil
}

func exec(conn *driver.Conn, stmt string) error {
	p, err := conn.Prepare(stmt)
	if err != nil {
		return errint.Wrap(err)
	}
	if err := p.Exec(); err != nil {
		return errint.Wrap(err)
	}
	if err := p.Close(); err != nil {
		return errint.Wrap(err)
	}
	return nil
}

func load(conn *driver.Conn, stmt string, args []string, f func(string) ([]*string, error)) error {
	if len(args) == 0 {
		return nil
	}

	p, err := conn.Prepare(stmt)
	if err != nil {
		return errint.Wrap(err)
	}
	load, err := p.Loader()
	if err != nil {
		return errint.Wrap(err)
	}

	for _, arg := range args {
		v, err := f(arg)
		if err != nil {
			return errint.Wrap(err)
		}
		if err := load.Load(v); err != nil {
			return errint.Wrap(err)
		}
	}

	return nil
}

func splitEnv(e string) (key, value string, err error) {
	v := strings.SplitN(e, "=", 2)
	if len(v) != 2 {
		return "", "", errint.Newf("building sys.env: expected k=v got %s", e)
	}
	return v[0], v[1], nil
}

//Close the sysdb and release
func (s *Sysdb) Close() error {
	return errint.Wrap(s.readAllEnv.Close())
}

//Environ dumps sys.env (which the user is free to modify)
//in Go readable format.
func (s *Sysdb) Environ() ([]string, error) {
	i, err := s.readAllEnv.Iter()
	if err != nil {
		return nil, errint.Wrap(err)
	}
	var env []string
	for i.Next() {
		r := i.Row()[0]
		env = append(env, *r)
	}
	if err := i.Err(); err != nil {
		return nil, errint.Wrap(err)
	}
	return env, nil
}
