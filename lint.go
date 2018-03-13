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

type Linter struct {
	v bool
	f *APKBUILD
}

func (l *Linter) Lint() {
	l.lintComments()
	l.lintMaintainer()
	l.lintContributers()
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

func (l *Linter) error(pos syntax.Pos, str string) {
	l.v = true // Linter found a style violation

	prefix := l.f.Name()
	if pos.IsValid() {
		prefix += ":" + pos.String()
	}

	fmt.Fprintf(os.Stderr, "%s: %s\n", prefix, str)
}
