package compile

import (
	"path/filepath"
	"strings"
	"unicode"

	"github.com/jimmyfrasche/etlite/internal/ast"
	"github.com/jimmyfrasche/etlite/internal/internal/errint"
	"github.com/jimmyfrasche/etlite/internal/virt"
)

const (
	inputDevice  = true
	outputDevice = false
)

func normFilename(nm string) string {
	base := filepath.Base(nm)
	idx := strings.LastIndexByte(base, '.')
	switch {
	case idx < 0:
		// filename
		nm = base
	case idx == 0:
		// .filename
		nm = base[1:]
	case idx > 0:
		// filename.ext
		nm = base[:idx]
	}
	//unlikely but remove any leading/trailing spaces or dots,
	//we don't care about any left in the middle though: up to user to escape
	return strings.TrimFunc(nm, func(r rune) bool {
		return r == '.' || unicode.IsSpace(r)
	})
}

func (c *compiler) derivedDeviceName(nm string) {
	c.frname = "" //old frame invalid on new device
	c.dname = nm
}

func (c *compiler) compileDevice(d ast.Device, read bool) {
	if d == nil {
		return
	}

	switch d := d.(type) {
	default:
		panic(errint.Newf("unrecognized Device type: %T", d))

	case *ast.DeviceStdio:
		if read {
			c.derivedDeviceName("-")
			c.push(virt.UseStdin())
		} else {
			c.push(virt.UseStdout())
		}

	case *ast.DeviceFile:
		name, ok := d.Name.Unescape()
		if !ok {
			panic(errint.Newf("file device name must be literal or string got %s", d.Name.Kind))
		}
		if read {
			c.derivedDeviceName(normFilename(name))
			c.push(virt.UseFileInput(name))
		} else {
			c.push(virt.UseFileOutput(name))
		}
	}
	if read {
		c.hadDevice = true
	}
}
