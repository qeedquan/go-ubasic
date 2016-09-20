package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/qeedquan/go-ubasic/interp"
)

var (
	status = 0
)

func main() {
	flag.Usage = usage
	flag.Parse()

	if flag.NArg() == 0 {
		ek(interp.Repl(interp.NewStdio(), os.Stdin))
	} else {
		for _, name := range flag.Args() {
			src, err := ioutil.ReadFile(name)
			if ek(err) {
				continue
			}
			ek(interp.Run(interp.NewStdio(), name, src))
		}
	}
	os.Exit(status)
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage: [file] ...")
	flag.PrintDefaults()
	os.Exit(2)
}

func ek(err error) bool {
	if err != nil {
		fmt.Fprintln(os.Stderr, "ubasic:", err)
		status = 1
		return true
	}
	return false
}
