// +build darwin dragonfly freebsd linux nacl netbsd openbsd solaris

package virt

import "runtime"

//TempDir returns the default directory to use for temporary files.
//
//This requires special handling on unix in case the env table's been altered.
func (m *Machine) TempDir() (string, error) {
	tmp, err := m.tmpFromEnv.Subquery()
	if err != nil {
		return "", err
	}
	if tmp != nil && *tmp != "" {
		return *tmp, nil
	}

	//This is adapted from the stdlib os's file_unix.go's definition of TempDir
	//which we can't call in case the TMPDIR environment variable is
	//set in the environment but was specifically deleted by a script.
	if runtime.GOOS == "android" {
		return "/data/local/tmp", nil
	}
	return "/tmp", nil
}
