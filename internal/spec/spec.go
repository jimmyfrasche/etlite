//Package spec defines the specifications for import and export from SQLite.
package spec

import (
	"github.com/jimmyfrasche/etlite/internal/builder"
	"github.com/jimmyfrasche/etlite/internal/conflict"
	"github.com/jimmyfrasche/etlite/internal/errint"
)

//Import specifies an import into SQLite.
type Import struct {
	Table     string          //the name of the created table, or "" for Internal tables
	Conflict  conflict.Method //Conflict method for insert
	Temporary bool            //if true created table is temporary (implicitly true if Internal)
	Header    []string        //if not nil, override header from input (must be same len)
	Internal  bool            //Internal tables are Temporary and have no name, regardless of other options
}

//Valid returns an error if s is not valid.
func (s Import) Valid() error {
	if s.requiresTable() {
		return errint.New("no name specified for table")
	}
	return s.Conflict.Valid()
}

func (s Import) requiresTable() bool {
	return s.Table == "" && !s.Internal
}

//Create produces SQL for creating the import table.
func (s Import) Create() string {
	b := builder.New("CREATE")

	if s.Temporary || s.Internal {
		b.Push("TEMPORARY")
	}

	b.Push("TABLE", s.Table, "(")
	b.CSV(s.Header, func(h string) {
		b.Push(h, "TEXT")
	})
	b.Push(")")

	return b.Join(" ")
}

//Insert produces SQL for inserting rows into the table.
func (s Import) Insert() string {
	b := builder.New("INSERT OR", s.Conflict.String(), "INTO", s.Table, "(")

	b.CSV(s.Header, func(h string) {
		b.Push(h)
	})

	b.Push(") VALUES (")

	b.CSV(s.Header, func(_ string) {
		b.Push("?")
	})
	b.Push(")")

	return b.Join(" ")
}

//Export specifies a export from SQLite.
type Export struct {
}

//Valid returns an error if s is invalid.
func (s Export) Valid() error {
	return nil
}
