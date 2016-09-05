package device

import (
	"runtime"
	"strings"
)

//misc hacks to work around the guarantees of the sys.env table.

//tempDir is like os.TempDir except it requires the value of
//the environment variable TEMPDIR to be passed in explicitly.
func tempDir(dir string) string {
	//ripped from os.TempDir() on 2016-02-07
	if dir == "" {
		if runtime.GOOS == "android" {
			dir = "/data/local/temp"
		} else {
			dir = "/tmp"
		}
	}
	return dir
}

//extractFromEnv finds which in env.
//Perhaps a little wasteful len(env) is rarely large enough
//to matter and the waste will be dwarfed by the IO surrounding
//this operation.
func extractFromEnv(which string, env []string) string {
	which += "="
	for _, e := range env {
		if strings.HasPrefix(e, which) {
			return e[len(which):]
		}
	}
	return ""
}
