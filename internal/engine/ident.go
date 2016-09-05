package engine

import "strings"

func uniqIdent(dflt string, idents []string, ignore map[string]bool) string {
	ln := 0
	prefixUsed := false
	for _, s := range idents {
		if ignore[s] {
			continue
		}
		if len(s) > ln {
			ln = len(s)
		}
		if strings.HasPrefix(s, dflt) {
			prefixUsed = true
		}
	}

	//dflt doesn't conflict with any used idents
	if ln < len(dflt) || !prefixUsed {
		return dflt
	}

	//create an ident as long as longest used, which will work
	//since it will have at least one digit appended
	return dflt + strings.Repeat("-", ln-len(dflt))
}
