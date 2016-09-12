package token

//Op is true if v is literal and first rune is one of "!@%&|-+=/<>"".
func (v Value) Op() bool {
	if v.Kind != Literal {
		return false
	}
	s := v.Value[0]
	for _, r := range "!@%&|-+=/<>*." {
		if byte(r) == s {
			return true
		}
	}
	return false
}

//Head is true if v is a literal whose canonical value is one of the list of
//valid literals for beginning a statement, as recognized by this SQLite superset.
func (v Value) Head(subquery bool) bool {
	if v.Kind != Literal {
		return false
	}
	lits := headLiterals[:]
	if subquery {
		lits = sqLiterals[:]
	}
	for _, h := range lits {
		if v.Canon == h {
			return true
		}
	}
	return false
}

var sqLiterals = [...]string{
	"IMPORT",
	"SELECT",
	"WITH",
}

var headLiterals = [...]string{
	"USE",
	"ASSERT",
	"DISPLAY",
	"IMPORT",
	"SELECT",
	"INSERT",
	"UPDATE",
	"DELETE",
	"REPLACE",
	"WITH",
	"CREATE",
	"DROP",
	"REINDEX",
	"ALTER",
	"VACUUM",
	"ATTACH",
	"DETACH",
	"PRAGMA",
	"COMMIT",
	"ROLLBACK",
	"SAVEPOINT",
	"RELEASE",

	"ANALYZE", //verboten for now but handled specially for better errors
	"EXPLAIN",
}
