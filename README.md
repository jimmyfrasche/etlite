super duper pre pre alpha.

extension of sqlite language with import/export

Brief nonnormative rundown of the syntax that is subject to change at whim and probably without updating this document. For the time being further information will have to refer to the source. (Plan to have it locked down with machine-checked documentation and examples for beta).

ETLite is a superset of a (large) subset of SQLite.

These additional statements are added:
- USE allows an ETLite script to specify an existing SQLite database to be the master db of the connection. (Must be first statement in script).
- DISPLAY [format] [TO device] allows changing the output format and IO redirection.
- IMPORT [TEMP|TEMPORARY] [format] [(col1, col2, ...)] [FROM device] [table] [limit] [offset] allows reading formatted data into a table.
- ASSERT message, subquery

Additionally, the @ placeholders work as follows: For @n where n is a natural number, this is the nth command line argument to the script or NULL. Otherwise @X refers to the environment variable X (or NULL if not set). Placeholders cannot be used in triggers.

Currently the formats are
- CSV [STRICT] [DELIM rune] [EOL DEFAULT|LF|UNIX|CRLF|WINDOWS] [NULL string] [HEADER bool]
- RAW [STRICT] [DELIM rune] [EOL DEFAULT|LF|UNIX|CRLF|WINDOWS] [NULL string] [HEADER bool]

NULL is a string used to indicate an SQL NULL value in the string output. If not set the empty string and NULL are the same.

RAW is CSV without a facility for quoting and `\t` as the default delimiter.

The device is either FILE filename or STDIN/STDOUT.

Any SQLite that returns rows is exported using the current DISPLAY settings.

As a statement, IMPORT creates a table and imports data into it.

The special form CREATE TABLE t (cols) FROM IMPORT imports data directly into t.

IMPORT may be used in most subqueries (outside of triggers), which creates and fills temporary tables, executes the desugared SQLite then drops the tables.

ASSERT ends the script if the scalar subquery returns anything other than 1 and prints message. If instead of a subquery an @ placeholder is given, it asserts the existence of that arg or env variable.

Otherwise, all SQLite is valid except for
- EXPLAIN/ANALYZE
- ROLLBACK (handled automatically (or implicitly but TBD automatically))
- placeholders (except @ which is handled differently as noted above)

SQLite is compiled with ICU/Rtree/FTS5/json/dbstat/soundex, a regexp function that links to PCRE, and the series, nextchar, and spellfix add-ons from ext/misc in the SQLite repo.
