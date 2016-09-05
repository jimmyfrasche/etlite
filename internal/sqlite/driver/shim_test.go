// +build !cgo

package driver

import (
	"flag"
	"log"
	"testing"
)

func TestMain(m *testing.M) {
	flag.Parse()
	log.Fatal("cannot run driver testing without cgo to link sqlite")
}
