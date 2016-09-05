#include "sqlite3.h"
#include "sqlite3ext.h"

/* I apologize for these macros but this is so repetitive */

#ifdef _WIN32
#	define win_decl __declspec(dllexport)
#else
#	define win_decl
#endif

#define def(F) win_decl extern int sqlite3_##F##_init(sqlite3*, char**, const sqlite3_api_routines*)

def(extension); //this is the regexp code
def(series);
def(nextchar);
def(spellfix);

#define reg(F) if((ret = sqlite3_auto_extension((void (*)(void))sqlite3_##F##_init)) != SQLITE_OK) goto error

int startup() {
	int ret = SQLITE_OK;
	reg(extension);
	reg(series);
	reg(nextchar);
	reg(spellfix);

error:
	return ret;
}

#undef reg
#undef def
#undef win_decl
