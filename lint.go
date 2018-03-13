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

type Linter struct {
	v bool
	f *APKBUILD
}

func (l *Linter) Lint() {
	l.lintComments()
	l.lintMaintainer()
	l.lintContributers()

	l.lintGlobalVariables()
	l.lintLocalVariables()
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

// lintMaintainer checks the global APKBUILD maintainer comment. It
// complains if there is not exactly one maintainer comment or if the
// maintainer comment is invalid.
func (l *Linter) lintMaintainer() {
	n := l.lintAddressComments(maintainerPrefix)
	if n == 0 {
		l.error(syntax.Pos{}, missingMaintainer)
	} else if n > 1 {
		l.error(syntax.Pos{}, tooManyMaintainers)
	}
}

// lintContributers checks the global APKBUILD contributor comments. It
// does the same job as lintMaintainer except that doesn't complain if
// none ore more than one contributor comments are found.
func (l *Linter) lintContributers() {
	l.lintAddressComments(contributorPrefix)
}

// lintGlobalVariables checks that all declared globally declared
// variables which are not prefixed with an underscore are metadata
// variables actually picked up by abuild(1). Besides it checks that all
// globally declared variables prefixed with an underscore are actually
// used.
func (l *Linter) lintGlobalVariables() {
	for _, a := range l.f.Assignments {
		v := a.Name.Value
		if !isMetaVar(v) {
			if v[0] != '_' {
				l.error(a.Pos(), invalidGlobalVar)
				continue
			}

			if l.f.UnusedVariable(v) {
				l.errorf(syntax.Pos{}, variableUnused, v)
			}
		}
	}

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
				if !(l.f.IsGlobalVar(v) || isMetaVar(v)) {
					l.errorf(x.Pos(), nonLocalVariable, v)
				}
			}

			return true
		}

		syntax.Walk(&f, fn)
	}
}

// lintAddressComments checks all global comments which start with given
// prefix followed by an ascii space character and makes sure that they
// contain a valid RFC 5322 mail address. It returns the amount of
// comment that started with the given prefix.
func (l *Linter) lintAddressComments(prefix string) int {
	var amount int
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

		sep := strings.Index(c.Text[1:], " ")
		if sep == -1 {
			l.error(c.Pos(), noAddressSeperator)
			continue
		}
		sep++ // first character of c.Text was skipped

		_, err := mail.ParseAddress(c.Text[sep+1:])
		if err != nil {
			l.error(c.Pos(), invalidAddress)
			continue
		}
	}

	return amount
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
