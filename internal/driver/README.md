Built with sqlite3 amalgamation version 3.14.1 using the following options:
SQLITE_THREADSAFE
SQLITE_ENABLE_DBSTAT_VTAB
SQLITE_ENABLE_RTREE
SQLITE_ENABLE_JSON1
SQLITE_ENABLE_FTS5
SQLITE_ENABLE_SOUNDEX
SQLITE_ENABLE_ICU

PCRE is dynamically linked against to support regexp.

Additionally ext/misc/{series,nextchar,spellfix}.c from the sqlite repo are required

sqlite3_mod_regexp.c is from github.com/mattn/go-sqlite3


For a different version you must rm shell.c before compiling or it will mess with cgo.
