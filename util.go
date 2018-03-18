package main

import (
	"mvdan.cc/sh/syntax"
	"os"
	"regexp"
)

var (
	// IsName checks if the given string is a valid name in the
	// shell command language as defined in section 3.235 of the
	// POSIX base specification.
	IsName = regexp.MustCompile("^[_A-Za-z0-9]+$").MatchString
)

// IsSpace reports whether the rune is an ascii space character. This
// differs from unicode.IsSpace which reports whether the rune is a
// space character as defined by Unicode's White Space property.
func IsSpace(r rune) bool {
	return r == ' '
}

// IsParamExp reports whether the given parameter expression can be
// replaced by a short parameter expression.
func IsParamExp(paramExp *syntax.ParamExp) bool {
	return paramExp.Excl || paramExp.Length || paramExp.Width ||
		paramExp.Index != nil || paramExp.Slice != nil ||
		paramExp.Repl != nil || paramExp.Exp != nil
}

// IsPrefixVar reports whether the given string is prefixed with a
// single ascii underscore character.
func IsPrefixVar(varname string) bool {
	if len(varname) < 2 {
		return false
	}

	return varname[0] == '_' && varname[1] != '_'
}

// IsIncluded reports whether the given string is included in the given
// string slice.
func IsIncluded(slice []string, str string) bool {
	for _, s := range slice {
		if s == str {
			return true
		}
	}

	return false
}

// IsMetaVar reports whether the given variable is a meta variable.
func IsMetaVar(varname string) bool {
	_, ok := metadataVariables[varname]
	return ok
}

// IsDir reports whether the given file name is a directory.
func IsDir(fn string) bool {
	fi, err := os.Stat(fn)
	return err == nil && fi.IsDir()
}

// Exists checks if a file with the given name exists.
func Exists(fn string) bool {
	_, err := os.Stat(fn)
	return !os.IsNotExist(err)
}
