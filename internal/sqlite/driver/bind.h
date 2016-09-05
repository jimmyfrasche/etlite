#ifndef CGO_SQLITE_BULK_INSERTER
#define CGO_SQLITE_BULK_INSERTER

#include "sqlite3.h"

//bulkRowsAtOnce defines how many rows we insert or read at once.
#define bulkRowsAtOnce 16

int sqlbind_subquery(sqlite3_stmt *, char **, int *);
int sqlbind_bulk_insert(sqlite3_stmt *, int, char **, int);
int sqlbind_bulk_read(sqlite3_stmt *, char ***, int *);

#endif
