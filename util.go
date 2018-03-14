package main

import (
	"mvdan.cc/sh/syntax"
	"os"
	"regexp"
)

var (
	IsName = regexp.MustCompile("^[_A-Za-z0-9]+$").MatchString
)

// IsSpace reports whether the rune is an ascii space character. This
// differs from unicode.IsSpace which reports whether the rune is a
// space character as defined by Unicode's White Space property.
func IsSpace(r rune) bool {
	return r == ' '
}

func IsParamExp(paramExp *syntax.ParamExp) bool {
	return paramExp.Excl || paramExp.Length || paramExp.Width ||
		paramExp.Index != nil || paramExp.Slice != nil ||
		paramExp.Repl != nil || paramExp.Exp != nil
}

func IsPrefixVar(varname string) bool {
	if len(varname) < 2 {
		return false
	}

	return varname[0] == '_' && varname[1] != '_'
}

func IsIncluded(slice []string, str string) bool {
	for _, s := range slice {
		if s == str {
			return true
		}
	}

	return false
}

func IsMetaVar(varname string) bool {
	return IsIncluded(metadataVariables, varname)
}

func IsDir(fn string) bool {
	fi, err := os.Stat(fn)
	return err == nil && fi.IsDir()
}

func Exists(fn string) bool {
	_, err := os.Stat(fn)
	return !os.IsNotExist(err)
}
