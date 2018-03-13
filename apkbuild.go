package main

import (
	"mvdan.cc/sh/syntax"
	"os"
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

	// Global variable assignments.
	Assignments []syntax.Assign

	// Declared functions.
	Functions map[string]syntax.FuncDecl
}

func Parse(fp string) (*APKBUILD, error) {
	parser := syntax.NewParser(syntax.KeepComments,
		syntax.Variant(lang))

	file, err := os.Open(fp)
	if err != nil {
		return nil, err
	}

	prog, err := parser.Parse(file, fp)
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
