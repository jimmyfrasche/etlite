package main

import (
	"flag"
	"io"
	"log"
	"os"
	"strings"

	"github.com/jimmyfrasche/etlite/internal/compile"
	"github.com/jimmyfrasche/etlite/internal/driver"
	"github.com/jimmyfrasche/etlite/internal/lex"
	"github.com/jimmyfrasche/etlite/internal/parse"
	"github.com/jimmyfrasche/etlite/internal/virt"
)

type autoClose struct { //TODO put in an internal package, change to io.ReadCloser
	f *os.File
}

func newAutoClose(f *os.File) *autoClose {
	return &autoClose{
		f: f,
	}
}

func (a *autoClose) Read(p []byte) (int, error) {
	if a.f == nil {
		return 0, io.EOF
	}
	n, err := a.f.Read(p)
	if err != nil {
		a.f.Close()
		a.f = nil
	}
	return n, err
}

func main() {
	log.SetFlags(0)

	var (
		srcFile = flag.String("f", "", "source file (defaults to stdin)")
		expr    = flag.String("e", "", "single expression")
	)
	flag.Parse()
	if *srcFile != "" && *expr != "" {
		flag.Usage()
		log.Fatal("-f and -e are mutually exclusive")
	}
	var (
		src       io.Reader
		name      string
		usesStdin bool
	)
	if *expr != "" {
		src = strings.NewReader(*expr)
		name = "<EXPR>"
	} else if *srcFile != "" && *srcFile != "/dev/stdin" { //force use of os.Stdin
		f, err := os.Open(*srcFile)
		if err != nil {
			log.Fatal(err)
		}
		src = newAutoClose(f)
		name = f.Name()
	} else {
		src = os.Stdin //XXX require this specifically be initiated by a flag?
		name = "<STDIN>"
		usesStdin = true
	}
	//XXX if above, and nothing selected, try first arg?

	err := driver.Init()
	if err != nil {
		log.Fatalln(err)
	}

	tokens := lex.Stream(name, src)
	nodes := parse.Tokens(tokens)
	db, bc, err := compile.Nodes(nodes, usesStdin) //TODO needs defined db, too?
	if err != nil {
		log.Fatal(err)
	}
	vm, err := virt.New(nil, virt.Spec{
		Database: db,
	})
	if err != nil {
		log.Fatal(err)
	}
	if err := vm.Run(bc); err != nil {
		log.Fatal(err)
	}
	errs := vm.Close()
	if len(errs) > 0 {
		log.Println("failed to shutdown database connection:")
		for err := range errs {
			log.Println(err)
		}
		os.Exit(1)
	}
}
