#ifndef CGO_SQLITE_BULK_INSERTER
#define CGO_SQLITE_BULK_INSERTER

#include "sqlite3.h"

//bulkRowsAtOnce defines how many rows we insert or read at once.
#define bulkRowsAtOnce 16

#define sqlbind_err_sqlite -1
#define sqlbind_err_num_cols -2
#define sqlbind_err_no_result -3
#define sqlbind_err_type -4
#define sqlbind_err_range -5
#define sqlbind_err_too_many_results -6

int sqlbind_assert_query(sqlite3 *, char *, int, int *);
int sqlbind_subquery(sqlite3_stmt *, char **, int *);
int sqlbind_bulk_insert(sqlite3_stmt *, int, char **, int);
int sqlbind_bulk_read(sqlite3_stmt *, char ***, int *);

#endif
