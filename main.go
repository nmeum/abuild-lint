package main

import (
	"fmt"
	"os"
	"path/filepath"
)

const (
	pkgbuildfn = "APKBUILD"
)

func main() {
	var fns []string
	if len(os.Args) <= 1 {
		if !Exists(pkgbuildfn) {
			fmt.Fprintf(os.Stderr, "%q doesn't exists in current directory.\n", pkgbuildfn)
			os.Exit(1)
		}

		fns = []string{pkgbuildfn}
	} else {
		for _, arg := range os.Args[1:] {
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
		linter := Linter{f: abuild, w: os.Stdout}
		if linter.Lint() {
			exitStatus = 1
		}
	}
	os.Exit(exitStatus)
}
