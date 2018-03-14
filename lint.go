package main

import (
	"fmt"
	"mvdan.cc/sh/syntax"
	"net/mail"
	"os"
	"strings"
)

const (
	// Prefix used to indicate that a comment specifies the package
	// maintainer.
	maintainerPrefix = " Maintainer:"

	// Prefix used to indicate that a comment specifies a package
	// contributor.
	contributorPrefix = " Contributor:"
)

// Array containing all variables which are directly used by
// abuild and should thus be declared globally.
var metadataVariables = []string{
	"pkgname",
	"pkgver",
	"pkgrel",
	"pkgdesc",
	"url",
	"arch",
	"license",
	"depends",
	"depends_dev",
	"makedepends",
	"checkdepends",
	"install",
	"subpackages",
	"source",
	"options",
	"patch_args",
	"builddir",
	"replaces",
	"md5sums",
	"sha256sums",
	"sha512sums",
	"install_if",
}

// Array containing all functions which can be declared by an APKBUILD
// and are then called from abuild(1). The function should be added in
// the order they are called by abuild(1).
var packageFunctions = []string{
	"snapshot",
	"sanitycheck",
	"fetch",
	"unpack",
	"prepare",
	"build",
	"check",
	"package",
}

type addressComment struct {
	c syntax.Comment
	a *mail.Address
}

type Linter struct {
	v bool
	f *APKBUILD
}

func (l *Linter) Lint() {
	l.lintComments()
	l.lintMaintainerAndContributors()

	l.lintGlobalVariables()
	l.lintGlobalCallExprs()
	l.lintLocalVariables()
	// TODO: check that $foo is used instead of ${foo} when possible
	// TODO: check that there are no empty lines between metadata assignments
	// XXX: maybe check that certain metadata variables are always declared
	// XXX: maybe check order of metadata variables

	// TODO: check that helper functions are prefixed with an _
	l.lintFunctionOrder()

	// TODO: check for forbidden bashisms
}

// lintComments checks that all comments start with a space. Shebangs
// are no exception to this rule since they shouldn't appear in an
// APKBUILD at all.
func (l *Linter) lintComments() {
	l.f.Walk(func(node syntax.Node) bool {
		c, ok := node.(*syntax.Comment)
		if ok && !strings.HasPrefix(c.Text, " ") {
			l.error(node.Pos(), badCommentPrefix)
		}

		return true
	})
}

// lintMaintainerAndContributors checks the APKBUILD maintainer and
// contributor comments. It complains if there is not exactly one
// maintainer comment, if the address specified in a maintainer or
// contributors comment doesn't conform to RFC 5322.
//
// Besides it checks that contributor comments are declared before
// maintainer comments and that contributor comments aren't declared
// twice. Regarding the order of the comments it also checks that the
// maintainer comment is declared before the first variable assignment.
func (l *Linter) lintMaintainerAndContributors() {
	var maintainer *addressComment
	n, m := l.lintAddressComments(maintainerPrefix)
	if n == 0 {
		l.error(syntax.Pos{}, missingMaintainer)
	} else if n > 1 {
		l.error(m[len(m)-1].c.Pos(), tooManyMaintainers)
	} else { // n == 1
		maintainer = &m[0]
	}

	if maintainer != nil && len(l.f.Assignments) > 0 &&
		maintainer.c.Pos().After(l.f.Assignments[0].Pos()) {
		l.error(maintainer.c.Pos(), maintainerAfterAssign)
	}

	addrMap := make(map[string]bool)
	_, contributors := l.lintAddressComments(contributorPrefix)
	for _, c := range contributors {
		pos := c.c.Pos()
		if maintainer != nil && pos.After(maintainer.c.Pos()) {
			l.error(pos, wrongAddrCommentOrder)
		}

		_, ok := addrMap[c.a.String()]
		if ok {
			l.error(pos, repeatedAddrComment)
		} else {
			addrMap[c.a.String()] = true
		}
	}

	// TODO: check for same address in contributor and maintainer?
}

// lintGlobalVariables checks that all declared globally declared
// variables which are not prefixed with an underscore are metadata
// variables actually picked up by abuild(1). Besides it checks that all
// globally declared variables prefixed with an underscore are actually
// used.
func (l *Linter) lintGlobalVariables() {
	for _, a := range l.f.Assignments {
		v := a.Name.Value
		if !IsMetaVar(v) {
			if !IsPrefixVar(v) {
				l.error(a.Pos(), invalidGlobalVar)
				continue
			}

			if l.f.IsUnusedVar(v) {
				l.errorf(syntax.Pos{}, variableUnused, v)
			}
		}
	}
}

// lintGlobalCallExprs check that all global shell statements don't use
// any kind of call expressions.
func (l *Linter) lintGlobalCallExprs() {
	l.f.Walk(func(node syntax.Node) bool {
		switch x := node.(type) {
		case *syntax.CallExpr:
			if len(x.Args) > 0 {
				l.error(node.Pos(), callExprInGlobalVar)
			}
		case *syntax.FuncDecl:
			return false
		}

		return true
	})
}

// lintLocalVariables checks that all variables declared inside a
// function are declared using the local keyword.
func (l *Linter) lintLocalVariables() {
	lvars := make(map[string][]string)
	for n, f := range l.f.Functions {
		fn := func(node syntax.Node) bool {
			switch x := node.(type) {
			case *syntax.DeclClause:
				if x.Variant.Value != "local" {
					return true
				}

				for _, a := range x.Assigns {
					lvars[n] = append(lvars[n], a.Name.Value)
				}
			case *syntax.Assign:
				for _, v := range lvars[n] {
					if v == x.Name.Value {
						return true
					}
				}

				v := x.Name.Value
				if !(l.f.IsGlobalVar(v) || IsMetaVar(v)) {
					l.errorf(x.Pos(), nonLocalVariable, v)
				}
			}

			return true
		}

		syntax.Walk(&f, fn)
	}
}

// lintFunctionOrder checks that all package functions are declared in
// the order they are called by abuild(1).
func (l *Linter) lintFunctionOrder() {
	var seen []*syntax.FuncDecl
	for _, fn := range packageFunctions {
		decl, ok := l.f.Functions[fn]
		if !ok {
			continue
		}

		for _, s := range seen {
			if !decl.Pos().After(s.Pos()) {
				l.errorf(decl.Pos(), wrongFuncOrder,
					decl.Name.Value, s.Name.Value)
			}
		}
		seen = append(seen, &decl)
	}

	// TODO: check subpackage functions
}

// lintAddressComments checks all global comments which start with given
// prefix followed by an ascii space character and makes sure that they
// contain a valid RFC 5322 mail address. It returns the amount of
// comment that started with the given prefix.
func (l *Linter) lintAddressComments(prefix string) (int, []addressComment) {
	var amount int
	var comments []addressComment

	for _, c := range l.f.Comments {
		if !strings.HasPrefix(c.Text, prefix) {
			continue
		}

		amount++
		if len(strings.TrimFunc(c.Text, IsSpace)) ==
			len(strings.TrimFunc(prefix, IsSpace)) {
			l.error(c.Pos(), missingAddress)
			continue
		}

		idx := len(prefix)
		if c.Text[idx] != ' ' {
			l.error(c.Pos(), noAddressSeperator)
			continue
		}

		a, err := mail.ParseAddress(c.Text[idx+1:])
		if err != nil {
			l.error(c.Pos(), invalidAddress)
			continue
		}

		comments = append(comments, addressComment{c, a})
	}

	return amount, comments
}

func (l *Linter) errorf(pos syntax.Pos, format string,
	argv ...interface{}) {
	l.v = true // Linter found a style violation

	prefix := l.f.Name()
	if pos.IsValid() {
		prefix += ":" + pos.String()
	}

	fmt.Fprintf(os.Stderr, "%s: %s\n", prefix,
		fmt.Sprintf(format, argv...))
}

func (l *Linter) error(pos syntax.Pos, str string) {
	l.errorf(pos, "%s", str)
}
