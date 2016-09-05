// +build windows plan9

package engine

import "os"

//TempDir returns the default directory to use for temporary files.
//
//This requires special handling on unix in case the env table's been altered.
func (m *Machine) TempDir() (string, error) {
	//this only needs special handling on unix
	return os.TempDir()
}
