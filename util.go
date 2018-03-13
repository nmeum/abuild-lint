package main

import (
	"os"
)

// IsSpace reports whether the rune is an ascii space character. This
// differs from unicode.IsSpace which reports whether the rune is a
// space character as defined by Unicode's White Space property.
func IsSpace(r rune) bool {
	return r == ' '
}

func IsDir(fn string) bool {
	fi, err := os.Stat(fn)
	return err == nil && fi.IsDir()
}

func Exists(fn string) bool {
	_, err := os.Stat(fn)
	return !os.IsNotExist(err)
}
