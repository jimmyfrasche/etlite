//Package parse parses a stream of lexemes into an ast.
//
//This is an island parser.
//That is, while it fully parses our extensions to SQLite,
//it largely accepts anything that vaguely resembles standard SQLite,
//letting the actual SQLite parser catch errors when it is executed.
//
//There are, however, a few heuristic checks to catch obviously marlformed SQLite.
package parse
