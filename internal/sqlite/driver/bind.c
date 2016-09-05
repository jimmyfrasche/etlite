#include <stdlib.h>
#include <assert.h>

#include "sqlite3.h"

#include "bind.h"

/* TODO really need to convert everything over to manual Pascal-style strings */

/*
 * sqlbind_subquery is the backend of the simulation of a subquery.
 * It assumes none of its arguments are nil and returns the result
 * into s and len.
 */
int sqlbind_subquery(sqlite3_stmt *p, char **s, int *len) {
	int rv = SQLITE_ERROR;
	if(p == NULL || s == NULL || len == NULL) {
		return rv;
	}

	rv = sqlite3_step(p);
	if(rv == SQLITE_DONE) {
		return rv;
	}

	*len = sqlite3_column_bytes(p, 0);
	*s = (char *) sqlite3_column_text(p, 0);

	return sqlite3_reset(p);
}

/*
 * sqlbind_bulk_insert takes a vector of strings
 * and repeatedly binds and executes p appropriately.
 *
 * It assumes that
 * - p is a valid prepared statement with at least one bind parameter
 * - p is properly reset and has no variables currently bound
 * - nbind divides nvars
 * - nbind, nvars > 0
 * - the binds are sequentially numbered and unnamed
 * - the length of vars is nvars
 * - the number of binds is nbinds
 * - the vector vars contains only null-terminated UTF-8 encoded strings or NULL
 * - each string can be safely freed after the call by stdlib free
 * - it is *not* responsible for savepointing or transactions
 *
 * It assures that
 * - the first error found aborts the process and is returned
 * - each string is freed
 * - the vector itself is freed
 * - resetting and rebinding of p is complete during each run
 * - C NULL entries in vars are assigned to SQL NULL
 */
int sqlbind_bulk_insert(sqlite3_stmt *p, int nbind, char **vars, int nvars) {
	assert(p != NULL);
	assert(vars != NULL);
	assert(nbind > 0);
	assert(nvars > 0);
	assert(nvars%nbind == 0);
	int rv = SQLITE_ERROR;
	int pos = 0;

	if(nvars == 0) {
		return rv;
	}

	/* Potential performance improvement: use pascal style strings
	 * so sqlite needn't strlen each arg, but hard to work around
	 * how cgo marshals Go strings to C.
	 * Likely more trouble than it's worth. */

	for(int step = 0; step < nvars/nbind; step++) {
		for(int n = 0; n < nbind; n++) {
			rv = sqlite3_bind_text(p, n+1, vars[pos], -1, &free); /* NB sqlite3_bind_text
																	given a NULL is the same as sqlite3_bind_null */
			if(rv != SQLITE_OK) {
				goto error;
			}
			pos++; /* need to keep track of consumed arguments in case of error */

		}
		rv = sqlite3_step(p);
		if(rv != SQLITE_DONE) {
			goto error;
		}

		rv = sqlite3_clear_bindings(p);
		if(rv != SQLITE_OK) {
			goto error;
		}

		rv = sqlite3_reset(p);
		if(rv != SQLITE_OK) {
			goto error;
		}
	}
	goto done;

error:
	/* need to clean up unused inputs */
	for(; pos < nvars; pos++) {
		if(vars+pos != NULL) {
			free(vars+pos);
		}
	}

done:
	free(vars);
	return rv;
}

/*
 * sqlbind_bulk_read reads a statement a row at a time.
 * It assumes p is valid and has columns to query.
 * */
int sqlbind_bulk_read(sqlite3_stmt *p, char ***vars, int *ncols) {
	int rv = sqlite3_step(p);
	if(rv == SQLITE_DONE || rv != SQLITE_ROW) {
		return rv;
	}

	*ncols = sqlite3_column_count(p);
	assert(*ncols > 0);

	*vars = (char**) malloc(*ncols * sizeof(char**));
	for(int i = 0; i < *ncols; i++) {
		(*vars)[i] = (char*) sqlite3_column_text(p, i);
	}

	return rv;
}
