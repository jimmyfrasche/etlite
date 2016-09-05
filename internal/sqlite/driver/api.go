//Package driver is a limited, specialized binding to a customized SQLite.
package driver

import "errors"

var (
	//NotImplemented is returned when this package is
	//compiled without cgo.
	NotImplemented = errors.New("not implemented")
)

//Init the sqlite engine and its extensions.
func Init() error {
	return startup()
}

//A Conn represents a connection to
//an underlying sqlite database.
type Conn struct {
	*conn
}

//Open db name.
func Open(name string) (*Conn, error) {
	c, err := open(name)
	if err != nil {
		return nil, err
	}
	return &Conn{c}, nil
}

//Close the db connection.
func (c *Conn) Close() error {
	return c.close()
}

//Prepare a query.
func (c *Conn) Prepare(query string) (*Stmt, error) {
	s, err := c.prepare(query)
	if err != nil {
		return nil, err
	}
	return &Stmt{s}, err
}

//A Stmt is a prepared query.
//Stmt exposes a number of APIs depending on the kind of query
//and its intended use.
//
//The most common is a statement that may return 0 or more results.
//For that there is Iter.
//
//A statement that loads into the database uses BulkLoad.
//
//Occasionally it's known that a statement returns no results,
//accepts no parameters, and is used only once.
//For that, there is Exec.
type Stmt struct {
	*stmt
}

//Close the prepared statement.
func (s *Stmt) Close() error {
	return s.close()
}

//String returns the string used to create this prepared statement.
func (s *Stmt) String() string {
	return s.string()
}

//Columns reports the columns returned by this query.
func (s *Stmt) Columns() []string {
	return s.columns()
}

//Exec executes a statement that takes no parameters and returns no results.
func (s *Stmt) Exec() error {
	return s.exec()
}

//Subquery emulates running the query in s as a subquery.
func (s *Stmt) Subquery() (*string, error) {
	return s.subquery()
}

//Loader creates a bulk loader.
func (s *Stmt) Loader() (*BulkLoader, error) {
	b, err := s.loader()
	if err != nil {
		return nil, err
	}
	return &BulkLoader{b}, err
}

//BulkLoader is a reverse iterator for shoveling data into SQLite.
type BulkLoader struct {
	*bulkLoader
}

//Load queues a row for loading and loads many rows in bulk
//when an internal limit is hit
func (b *BulkLoader) Load(vs []*string) error {
	return b.load(vs)
}

//Close flushes any remaining rows.
//It does not Close the underlying prepared statement.
func (b *BulkLoader) Close() error {
	//TODO(jmf) rename Flush
	return b.close()
}

//Iter returns an iterator over the results of the query.
func (s *Stmt) Iter() (*Iter, error) {
	i, err := s.iter()
	if err != nil {
		return nil, err
	}
	return &Iter{i}, nil
}

//Iter represents iterator over the results of a query.
type Iter struct {
	*iter
}

//Next reports whether there is a next result.
func (i *Iter) Next() bool {
	return i.next()
}

//Row returns a copy of the current row.
func (i *Iter) Row() []*string {
	return i.row()
}

//Err reports any error encountered during iteration.
func (i *Iter) Err() error {
	return i.error()
}
