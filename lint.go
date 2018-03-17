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

type metaPos int

const (
	beforeFuncs metaPos = iota
	afterFuncs
)

// Array containing all variables which are directly used by
// abuild and should thus be declared globally.
var metadataVariables = map[string]metaPos{
	"pkgname":      beforeFuncs,
	"pkgver":       beforeFuncs,
	"pkgrel":       beforeFuncs,
	"pkgdesc":      beforeFuncs,
	"url":          beforeFuncs,
	"arch":         beforeFuncs,
	"license":      beforeFuncs,
	"depends":      beforeFuncs,
	"depends_dev":  beforeFuncs,
	"makedepends":  beforeFuncs,
	"checkdepends": beforeFuncs,
	"install":      beforeFuncs,
	"subpackages":  beforeFuncs,
	"source":       beforeFuncs,
	"options":      beforeFuncs,
	"patch_args":   beforeFuncs,
	"builddir":     beforeFuncs,
	"replaces":     beforeFuncs,
	"install_if":   beforeFuncs,
	"md5sums":      afterFuncs,
	"sha256sums":   afterFuncs,
	"sha512sums":   afterFuncs,
}

// Array containing all functions which can be declared by an APKBUILD
// and are then called from abuild(1). The function should be added in
// the order they are called by abuild(1).
var packageFunctions = []string{
	"sanitycheck",
	"snapshot",
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

func (l *Linter) Lint() bool {
	l.lintComments()
	l.lintMaintainerAndContributors()

	l.lintGlobalVariables()
	l.lintGlobalCmdSubsts()
	l.lintLocalVariables()
	l.lintUnusedVariables()
	l.lintParamExpression()
	// TODO: check for required metadata variables (pkgname, pkgurl, …)
	// TODO: check that there are no empty lines between metadata assignments
	// XXX: maybe check that certain metadata variables are always declared
	l.lintMetadataPlacement()

	// TODO: check that helper functions are prefixed with an _
	l.lintFunctionOrder()

	// TODO: check for space between last function declaration and checksum

	// TODO: check for forbidden bashisms

	return l.v
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
	}

	if len(m) >= 1 {
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
// variables actually picked up by abuild(1).
func (l *Linter) lintGlobalVariables() {
	for _, a := range l.f.Assignments {
		v := a.Name.Value
		if !IsMetaVar(v) && !IsPrefixVar(v) {
			l.error(a.Pos(), invalidGlobalVar)
			continue
		}
	}
}

// lintUnusedVariables checks if all globally and locally declared
// non-metadata variable are actually used somewhere in the APKBUILD.
func (l *Linter) lintUnusedVariables() {
	l.f.Walk(func(node syntax.Node) bool {
		assign, ok := node.(*syntax.Assign)
		if !ok {
			return true
		}

		v := assign.Name.Value
		if !IsMetaVar(v) && l.f.IsUnusedVar(v) {
			l.errorf(assign.Pos(), variableUnused, v)
		}

		return true
	})
}

// lintGlobalCmdSubsts check that all global shell statements don't use
// any kind of command substitutions.
func (l *Linter) lintGlobalCmdSubsts() {
	l.f.Walk(func(node syntax.Node) bool {
		switch node.(type) {
		case *syntax.CmdSubst:
			l.error(node.Pos(), cmdSubstInGlobalVar)
		case *syntax.FuncDecl:
			return false
		}

		return true
	})
}

// lintLocalVariables checks that all variables declared inside a
// function are declared using the local keyword.
func (l *Linter) lintLocalVariables() {
	vars := make(map[string][]string)
	for n, f := range l.f.Functions {
		fn := func(node syntax.Node) bool {
			switch x := node.(type) {
			case *syntax.DeclClause:
				variant := x.Variant.Value
				if variant != "local" && variant != "export" {
					return true
				}

				for _, a := range x.Assigns {
					vars[n] = append(vars[n], a.Name.Value)
				}
			case *syntax.WordIter:
				if l.isValidVarScope(vars[n], x.Name) {
					return true
				}
			case *syntax.Assign:
				if l.isValidVarScope(vars[n], x.Name) {
					return true
				}
			}

			return true
		}

		syntax.Walk(&f, fn)
	}
}

// lintParamExpression checks for long parameter expansion with the form
// ${…} and checks if they can be replaced by a semantically equivalent
// short parameter expansion with a $… form.
func (l *Linter) lintParamExpression() {
	var words []*syntax.Word
	l.f.Walk(func(node syntax.Node) bool {
		word, ok := node.(*syntax.Word)
		if ok {
			words = append(words, word)
			return false
		}

		return true
	})

	for _, word := range words {
		nparts := len(word.Parts)
		for n, p := range word.Parts {
			paramExp, ok := p.(*syntax.ParamExp)
			if !ok {
				continue
			} else if paramExp.Short {
				continue
			} else if IsParamExp(paramExp) {
				continue
			}

			if n < nparts-1 {
				next := word.Parts[n+1]
				lit, ok := next.(*syntax.Lit)
				if !ok || IsName(lit.Value) {
					continue
				}
			}

			l.errorf(paramExp.Pos(), trivialLongParamExp,
				paramExp.Param.Value, paramExp.Param.Value)
		}
	}

}

// lintMetadataPlacement checks the placement of metadata variables.
// Some need to be declared before the first function declaration,
// others need to be declared after the last function declaration.
func (l *Linter) lintMetadataPlacement() {
	var firstFn, lastFn *syntax.FuncDecl
	for _, fn := range l.f.Functions {
		if firstFn == nil || !fn.Pos().After(firstFn.Pos()) {
			firstFn = &fn
		}
		if lastFn == nil || fn.Pos().After(lastFn.Pos()) {
			lastFn = &fn
		}
	}

	for _, v := range l.f.Assignments {
		name := v.Name.Value
		mpos, ok := metadataVariables[name]
		if !ok {
			continue
		}

		vpos := v.Pos()
		switch mpos {
		case beforeFuncs:
			if firstFn != nil && vpos.After(firstFn.Pos()) {
				l.errorf(vpos, metadataBeforeFunc, name)
			}
		case afterFuncs:
			if lastFn != nil && !vpos.After(lastFn.Pos()) {
				l.errorf(vpos, metadataAfterFunc, name)
			}
		}
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
			l.error(c.Pos(), noAddressSeparator)
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

func (l *Linter) isValidVarScope(vars []string, v *syntax.Lit) bool {
	if IsIncluded(vars, v.Value) {
		return true
	}

	if !(l.f.IsGlobalVar(v.Value) || IsMetaVar(v.Value)) {
		l.errorf(v.Pos(), nonLocalVariable, v.Value)
		return false
	}

	return true
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
