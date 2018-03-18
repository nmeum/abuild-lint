package main

import (
	"io"
	"mvdan.cc/sh/syntax"
)

const (
	// Shell variant to use. Even though APKBUILDs are mostly
	// written in POSIX shell we still use LangBash here to check
	// for the `local` variable declaration keyword and other
	// bashims permitted in APKBUILDs.
	lang = syntax.LangBash
)

type APKBUILD struct {
	// Root node of the AST.
	prog *syntax.File

	// Globally declared comments.
	Comments []syntax.Comment

	// Global variable assignments excluding environment variables.
	Assignments []syntax.Assign

	// Declared functions.
	Functions map[string]syntax.FuncDecl
}

func Parse(r io.Reader, name string) (*APKBUILD, error) {
	parser := syntax.NewParser(syntax.KeepComments,
		syntax.Variant(lang))

	prog, err := parser.Parse(r, name)
	if err != nil {
		return nil, err
	}

	apkbuild := APKBUILD{prog: prog}
	apkbuild.Functions = make(map[string]syntax.FuncDecl)
	apkbuild.Walk(apkbuild.visit)

	return &apkbuild, nil
}

func (a *APKBUILD) Name() string {
	return a.prog.Name
}

func (a *APKBUILD) Walk(f func(syntax.Node) bool) {
	syntax.Walk(a.prog, f)
}

func (a *APKBUILD) visit(node syntax.Node) bool {
	switch x := node.(type) {
	case *syntax.DeclClause:
		return x.Variant.Value != "export"
	case *syntax.FuncDecl:
		a.Functions[x.Name.Value] = *x
		return false // All nodes after this have local scope
	case *syntax.Assign:
		a.Assignments = append(a.Assignments, *x)
		return true
	case *syntax.Comment:
		a.Comments = append(a.Comments, *x)
		return true
	default:
		return true
	}
}

func (a *APKBUILD) IsGlobalVar(varname string) bool {
	for _, assignment := range a.Assignments {
		if assignment.Name.Value == varname {
			return true
		}
	}

	return false
}

func (a *APKBUILD) IsUnusedVar(varname string) bool {
	ret := true
	a.Walk(func(node syntax.Node) bool {
		switch x := node.(type) {
		case *syntax.DeclClause:
			if x.Variant.Value != "export" {
				return true
			}

			for _, a := range x.Assigns {
				if a.Name.Value == varname {
					ret = false
					return false
				}
			}
		case *syntax.SglQuoted:
			if x.Dollar && x.Value == varname {
				ret = false
				return false
			}
		case *syntax.ParamExp:
			if x.Param.Value == varname {
				ret = false
				return false
			}
		}

		return true
	})

	return ret
}
