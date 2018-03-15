package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
)

const (
	pkgbuildfn = "APKBUILD"
)

func main() {
	flag.Parse()

	var fns []string
	if flag.NArg() == 0 {
		if !Exists(pkgbuildfn) {
			fmt.Fprintf(os.Stderr, "%q doesn't exists in current directory.\n", pkgbuildfn)
			os.Exit(1)
		}

		fns = []string{pkgbuildfn}
	} else {
		for _, arg := range flag.Args() {
			if IsDir(arg) {
				arg = filepath.Join(arg, pkgbuildfn)
			}

			if !Exists(arg) {
				fmt.Fprintf(os.Stderr, "%q doesn't exist.\n", arg)
				os.Exit(1)
			}

			fns = append(fns, arg)
		}
	}

	var abuilds []*APKBUILD
	for _, fn := range fns {
		file, err := os.Open(fn)
		if err != nil {
			panic(err)
		}

		abuild, err := Parse(file, fn)
		if err != nil {
			panic(err)
		}

		abuilds = append(abuilds, abuild)
		file.Close()
	}

	exitStatus := 0
	for _, abuild := range abuilds {
		linter := Linter{f: abuild}
		if linter.Lint() {
			exitStatus = 1
		}
	}
	os.Exit(exitStatus)
}
